package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"gopkg.in/op/go-logging.v1"
)

const (
	environmentConfigPrefix = "env."
)

var (
	registeredTypes = make(map[string]MuninType)

	noMultigraph = ""

	// ErrSkippedOutput is reported when output was skipped (such as blank lines)
	// It should never reach code outside of this package
	ErrSkippedOutput = errors.New("Skipped output")

	// ErrCommandTimeout occurrs when a command took too long to run and had to be
	// killed
	ErrCommandTimeout = errors.New("Command timeout")
)

// MuninState holds the current state of a munin plugin
type MuninState struct {
	typeHandler   MuninType
	hasPrevious   bool
	previousValue float64
	previousTime  time.Time
}

// MuninRunner holds the necessary information to run a munin plugin
type MuninRunner struct {
	scriptPath    string
	name          string
	user          string
	group         string
	waitGroup     sync.WaitGroup
	additionalEnv map[string]string
	commandTimer  *time.Timer

	isConfigured   bool
	multigraph     *string
	collectionTime time.Time
	state          map[string]*MuninState
	data           map[string]interface{}
}

// NewMuninRunner creates a new runner for the given munin plugin script
func NewMuninRunner(scriptPath, name string) *MuninRunner {
	m := &MuninRunner{
		scriptPath:    scriptPath,
		name:          name,
		user:          "nobody",
		group:         "nobody",
		additionalEnv: map[string]string{},
		commandTimer:  time.NewTimer(0),

		state: map[string]*MuninState{},
	}

	// Empty the timer
	<-m.commandTimer.C

	return m
}

// ApplySection applies a configuration file section to the runner so it can set
// up necessary run user/group and environment variables
// This must be called BEFORE Configure or a panic occurs
func (m *MuninRunner) ApplySection(section map[string]string) error {
	if m.isConfigured {
		panic("ApplySection was called after Configure was completed")
	}

	for k, v := range section {
		if k == "user" {
			m.user = v
			continue
		}

		if k == "group" {
			m.group = v
			continue
		}

		if strings.HasPrefix(k, environmentConfigPrefix) {
			m.additionalEnv[k[len(environmentConfigPrefix):]] = v
			continue
		}

		// Ignore
	}

	return nil
}

// Name returns the munin plugin name
func (m *MuninRunner) Name() string {
	return m.name
}

// Configure runs the munin plugin config command in order to setup recognised
// fields and types
// After Configure is called, ApplySection can no longer be run, and Collect can
// now be run
func (m *MuninRunner) Configure(cache *CredentialCache) error {
	// Run config param to grab the types
	m.multigraph = &noMultigraph
	if err := m.runScript(cache, "config", (*MuninRunner).handleConfigLine); err != nil {
		return err
	}

	if log.IsEnabledFor(logging.DEBUG) {
		for field, state := range m.state {
			log.Debug("[%s] Field %s has type %s", m.name, field, state.typeHandler.Name())
		}
	}

	m.isConfigured = true

	return nil
}

// Collect runs the munin plugin and collects the results
// Configure must have completed before running Collect
func (m *MuninRunner) Collect(cache *CredentialCache, collectionTime time.Time) (map[string]interface{}, error) {
	if !m.isConfigured {
		panic("Runner was not Configured")
	}

	m.multigraph = &noMultigraph
	m.collectionTime = collectionTime
	m.data = make(map[string]interface{})

	if err := m.runScript(cache, "", (*MuninRunner).handleOutputLine); err != nil {
		return nil, err
	}

	return m.data, nil
}

// runScript runs the munin plugin with the given parameters and passes each
// output line to the given callback
func (m *MuninRunner) runScript(cache *CredentialCache, param string, callback func(*MuninRunner, []byte) error) error {
	command, err := m.initCommand(cache, param)
	if err != nil {
		return err
	}

	stdoutPipe, err := command.StdoutPipe()
	if err != nil {
		return err
	}

	stderrPipe, err := command.StderrPipe()
	if err != nil {
		return err
	}

	if err := command.Start(); err != nil {
		return err
	}

	finishChan := make(chan error, 2)
	m.waitGroup.Add(2)

	go m.lineReader(stdoutPipe, finishChan, callback)
	go m.lineReader(stderrPipe, finishChan, (*MuninRunner).handleErrorLine)

	if err := m.processHandler(command, finishChan); err != nil {
		command.Process.Kill()
	}

	command.Wait()
	m.waitGroup.Wait()

	if err != nil {
		if param == "" {
			log.Errorf("[%s] Script failed: %s", m.name, err)
		} else {
			log.Errorf("[%s] Script failed for %s: %s", m.name, param, err)
		}
		return err
	}

	return nil
}

// initCommand initialises a new exec.Cmd structure to run the munin plugin with
func (m *MuninRunner) initCommand(cache *CredentialCache, param string) (*exec.Cmd, error) {
	command := &exec.Cmd{}

	command.SysProcAttr = &syscall.SysProcAttr{}
	command.SysProcAttr.Credential = &syscall.Credential{}
	if err := m.setCredentials(command, cache); err != nil {
		return nil, err
	}

	command.Path = path.Join(m.scriptPath, m.name)
	if param != "" {
		command.Args = []string{command.Path, param}
	}

	log.Debug("[%s] Path: %s", m.name, command.Path)
	log.Debug("[%s] Args: %v", m.name, command.Args)

	m.applyAdditionalEnv(command)
	m.applyPluginEnv(command)

	if log.IsEnabledFor(logging.DEBUG) {
		for _, v := range command.Env {
			log.Debug("[%s] Env: %s", m.name, v)
		}
	}

	return command, nil
}

// setCredentials sets the run user and group for the given exec.Cmd
func (m *MuninRunner) setCredentials(command *exec.Cmd, cache *CredentialCache) error {
	if err := m.applyCommandUser(command, m.user, cache); err != nil {
		return err
	}

	if err := m.applyCommandGroup(command, m.group, cache); err != nil {
		return err
	}

	return nil
}

// processHandler monitors the exec.Cmd and applies a timeout to it, killing the
// process if it does not complete in the given time
func (m *MuninRunner) processHandler(command *exec.Cmd, finishChan <-chan error) (err error) {
	// Reset the timer
	sawTimeout := false
	// TODO: Make configurable?
	m.commandTimer.Reset(10 * time.Second)

	c := 0
FinishLoop:
	for c != 2 {
		select {
		case result := <-finishChan:
			if result != nil {
				err = result
				break FinishLoop
			}
			c++
		case <-m.commandTimer.C:
			err = ErrCommandTimeout
			sawTimeout = true
			break
		}
	}

	// Stop and clear any stale trigger
	if !m.commandTimer.Stop() && !sawTimeout {
		<-m.commandTimer.C
	}

	return
}

// applyEnv adds an environment variable to the command
func (m *MuninRunner) applyEnv(command *exec.Cmd, name string, value string) {
	// Validate. This needs checking
	// I'm guessing here from some things I read about IEEE Std 1003.1-2001
	name = strings.Replace(name, "\x00", "", -1)
	name = strings.Replace(name, "=", "", -1)

	command.Env = append(command.Env, name+"="+value)
}

// applyAdditionalEnv adds additional environment variables set in the munin
// plugin configuration to the command
func (m *MuninRunner) applyAdditionalEnv(command *exec.Cmd) {
	for k, v := range m.additionalEnv {
		m.applyEnv(command, k, v)
	}
}

// applyPluginEnv sets environment variables for the munin plugin in the same
// way that munin-node does
// This includes the plugin state paths and other munin variabels
func (m *MuninRunner) applyPluginEnv(command *exec.Cmd) {
	m.applyEnv(command, "MUNIN_MASTER_IP", "-")
	m.applyEnv(command, "MUNIN_CAP_MULTIGRAPH", "1")

	// TODO: Determine this from the munin-node.conf? Can it be changed there?
	m.applyEnv(command, "MUNIN_LIBDIR", "/usr/share/munin")

	m.applyEnv(command, "MUNIN_PLUGSTATE", "/var/lib/munin/plugin-state/"+m.user)
	m.applyEnv(command, "MUNIN_STATEFILE", "/var/lib/munin/plugin-state/"+m.user+"/"+m.name+"-")
}

// getFieldIdx calculates our internal unique name for a field, as multigraph
// functionality means we can have multiple with the same field name
func (m *MuninRunner) getFieldIdx(field string) string {
	if m.multigraph == &noMultigraph {
		return field
	}

	return *m.multigraph + "." + field
}

// handleConfigLine processes a single line of the plugin config output
func (m *MuninRunner) handleConfigLine(line []byte) error {
	names, value, err := m.parseLine(line)
	if err == ErrSkippedOutput {
		return nil
	} else if err != nil {
		return err
	}

	if len(names) == 1 && names[0] == "multigraph" {
		*m.multigraph = string(value)
		return nil
	}

	idx := m.getFieldIdx(names[0])

	if _, hasState := m.state[idx]; !hasState {
		m.state[idx] = &MuninState{}
	}

	// Ignore anything but .type entries
	if len(names) != 2 || names[1] != "type" {
		// Default to using GAUGE if we don't have an entry for this field yet
		if m.state[idx].typeHandler == nil {
			m.state[idx].typeHandler = registeredTypes[defaultType]
		}
		return nil
	}

	typeHandler, ok := registeredTypes[value]
	if !ok {
		// Skip unsupported types
		log.Warning("[%s] Field %s has unsupported field type %s", m.name, idx, value)
		return nil
	}

	m.state[idx].typeHandler = typeHandler
	return nil
}

// handleOutputLine process a single line of the plugin output
func (m *MuninRunner) handleOutputLine(line []byte) error {
	names, value, err := m.parseLine(line)
	if err == ErrSkippedOutput {
		return nil
	} else if err != nil {
		return err
	}

	if len(names) == 1 && names[0] == "multigraph" {
		*m.multigraph = string(value)
		return nil
	}

	if len(names) != 2 || names[1] != "value" {
		// Ignore anything but .value entries
		return nil
	}

	var idx string
	if m.multigraph == &noMultigraph {
		idx = names[0]
	} else {
		idx = *m.multigraph + "." + names[0]
	}

	state, ok := m.state[idx]
	if !ok {
		return fmt.Errorf("Unknown field: %s", idx)
	}

	if string(value) == "U" {
		log.Debug("[%s] Field %s skipped, value is U (unknown)", m.name, idx)
		return nil
	}

	floatValue, err := strconv.ParseFloat(string(value), 64)
	if err != nil {
		return err
	}

	var duration float64
	hasPrevious := state.hasPrevious
	previousValue := state.previousValue

	if hasPrevious {
		duration = float64(m.collectionTime.Sub(state.previousTime))
	}

	state.hasPrevious = true
	state.previousValue = floatValue
	state.previousTime = m.collectionTime
	if !hasPrevious {
		if state.typeHandler.RequiresPrevious() {
			log.Debug("[%s] Field %s skipped this once as its type requires a previous value", m.name, idx)
			return nil
		}
	} else if duration <= 0 {
		return nil
	}

	eventFieldPrefix := "munin_" + m.name + "_"
	m.data[eventFieldPrefix+idx] = state.typeHandler.Calculate(floatValue, previousValue, duration)
	log.Debug("[%s] Field %s: %f", m.name, idx, m.data[eventFieldPrefix+idx])

	return nil
}

// handleErrorLine processes the stderr pipe for processes and reports all
// output on this pipe to the logs
func (m *MuninRunner) handleErrorLine(line []byte) error {
	log.Warning("[%s] Stderr: %s", m.name, line)
	return nil
}

// lineReader reads lines from the given reader and calls the given callback
// for each line
func (m *MuninRunner) lineReader(reader io.ReadCloser, finishChan chan<- error, callback func(*MuninRunner, []byte) error) {
	defer func() {
		m.waitGroup.Done()
	}()

	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		if err := callback(m, scanner.Bytes()); err != nil {
			log.Errorf("[%s] Output parsing failed: %s", m.name, err.Error())
			finishChan <- err
			return
		}
	}

	finishChan <- nil
}

// parseLine parses a line of plugin output into a slice of dot-separate names
// and a value
func (m *MuninRunner) parseLine(line []byte) ([]string, string, error) {
	if len(line) == 0 {
		return nil, "", ErrSkippedOutput
	}

	line = bytes.Trim(line, " \t")
	if len(line) == 0 {
		return nil, "", ErrSkippedOutput
	}

	if line[0] == '#' {
		return nil, "", ErrSkippedOutput
	}

	split := bytes.IndexAny(line, " \t")
	if split == -1 {
		return nil, "", fmt.Errorf("Invalid output: %s", line)
	}

	name := string(line[:split])
	value := bytes.TrimLeft(line[split+1:], " \t")

	names := strings.Split(name, ".")
	return names, string(value), nil
}
