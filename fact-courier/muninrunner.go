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

	ErrSkippedOutput  = errors.New("Skipped output")
	ErrCommandTimeout = errors.New("Command timeout")
)

type MuninRunner struct {
	scriptPath    string
	name          string
	user          string
	group         string
	waitGroup     sync.WaitGroup
	additionalEnv map[string]string
	commandTimer  *time.Timer

	multigraph   *string
	types        map[string]MuninType
	data         map[string]interface{}
	previousData map[string]float64
}

func NewMuninRunner(scriptPath, name string) *MuninRunner {
	m := &MuninRunner{
		scriptPath:    scriptPath,
		name:          name,
		user:          "nobody",
		group:         "nobody",
		additionalEnv: map[string]string{},
		commandTimer:  time.NewTimer(time.Second),

		types:        map[string]MuninType{},
		previousData: map[string]float64{},
	}

	// Stop timer and be sure to clear stale entry
	if !m.commandTimer.Stop() {
		<-m.commandTimer.C
	}

	return m
}

func (m *MuninRunner) ApplySection(section map[string]string) error {
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

func (m *MuninRunner) Name() string {
	return m.name
}

func (m *MuninRunner) Configure(cache *CredentialCache) error {
	// Run config param to grab the types
	m.multigraph = &noMultigraph
	if err := m.runScript(cache, "config", (*MuninRunner).handleConfigLine); err != nil {
		return err
	}

	return nil
}

func (m *MuninRunner) Collect(cache *CredentialCache) (map[string]interface{}, error) {
	m.data = make(map[string]interface{})
	m.multigraph = &noMultigraph

	if err := m.runScript(cache, "", (*MuninRunner).handleOutputLine); err != nil {
		return nil, err
	}

	return m.data, nil
}

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

	m.applyEnv(command, "MUNIN_MASTER_IP", "-")
	m.applyEnv(command, "MUNIN_CAP_MULTIGRAPH", "1")
	m.applyEnv(command, "MUNIN_LIBDIR", "/usr/share/munin")
	m.applyAdditionalEnv(command)
	m.applyPluginStateEnv(command)

	if log.IsEnabledFor(logging.DEBUG) {
		for _, v := range command.Env {
			log.Debug("[%s] Env: %s", m.name, v)
		}
	}

	return command, nil
}

func (m *MuninRunner) setCredentials(command *exec.Cmd, cache *CredentialCache) error {
	if err := m.applyCommandUser(command, m.user, cache); err != nil {
		return err
	}

	if err := m.applyCommandGroup(command, m.group, cache); err != nil {
		return err
	}

	return nil
}

func (m *MuninRunner) processHandler(command *exec.Cmd, finishChan <-chan error) (err error) {
	// Reset the timer
	sawTimeout := false
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

func (m *MuninRunner) applyEnv(command *exec.Cmd, name string, value string) {
	// Validate. This needs checking
	// I'm guessing here from some things I read about IEEE Std 1003.1-2001
	name = strings.Replace(name, "\x00", "", -1)
	name = strings.Replace(name, "=", "", -1)

	command.Env = append(command.Env, name+"="+value)
}

func (m *MuninRunner) applyAdditionalEnv(command *exec.Cmd) {
	for k, v := range m.additionalEnv {
		m.applyEnv(command, k, v)
	}
}

func (m *MuninRunner) applyPluginStateEnv(command *exec.Cmd) {
	m.applyEnv(command, "MUNIN_PLUGSTATE", "/var/lib/munin/plugin-state/"+m.user)
	m.applyEnv(command, "MUNIN_STATEFILE", "/var/lib/munin/plugin-state/"+m.user+"/"+m.name+"-")
}

func (m *MuninRunner) getFieldIdx(field string) string {
	if m.multigraph == &noMultigraph {
		return field
	}

	return *m.multigraph + "." + field
}

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

	// Ignore anything but .type entries
	if len(names) != 2 || names[1] != "type" {
		// Default to using GAUGE if we don't have an entry for this field yet
		if _, hasType := m.types[idx]; !hasType {
			m.types[idx] = registeredTypes[defaultType]
		}
		return nil
	}

	typeHandler, ok := registeredTypes[value]
	if !ok {
		// Skip unsupported types
		log.Warning("[%s] Field %s has unsupported field type %s", m.name, idx, value)

		return nil
	}

	log.Debug("[%s] Field %s has type %s", m.name, idx, value)
	m.types[idx] = typeHandler

	return nil
}

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

	floatValue, err := strconv.ParseFloat(string(value), 64)
	if err != nil {
		return err
	}

	var idx string
	if m.multigraph == &noMultigraph {
		idx = names[0]
	} else {
		idx = *m.multigraph + "." + names[0]
	}

	typeHandler, ok := m.types[idx]
	if !ok {
		return fmt.Errorf("Unknown field: %s", idx)
	}

	previousValue, hasPrevious := m.previousData[idx]
	m.previousData[idx] = floatValue
	if !hasPrevious && typeHandler.RequiresPrevious() {
		return nil
	}

	m.data["value_"+idx] = typeHandler.Calculate(floatValue, previousValue)
	log.Debug("[%s] Field %s: %f", m.name, idx, m.data["value_"+idx])

	return nil
}

func (m *MuninRunner) handleErrorLine(line []byte) error {
	log.Warning("[%s] Stderr: %s", m.name, line)
	return nil
}

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
