package main

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	environmentConfigPrefix = "env."

	ErrInvalidOutput  = errors.New("Invalid output")
	ErrCommandTimeout = errors.New("Command timeout")
)

type MuninRunner struct {
	scriptPath    string
	name          string
	user          string
	group         string
	additionalEnv map[string]string
	commandTimer  *time.Timer
	outputChan    chan interface{}
	waitGroup     sync.WaitGroup
	output        []byte
}

func NewMuninRunner(scriptPath, name string) *MuninRunner {
	m := &MuninRunner{
		scriptPath:    scriptPath,
		name:          name,
		user:          "nobody",
		group:         "nobody",
		commandTimer:  time.NewTimer(time.Second),
		additionalEnv: map[string]string{},
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

func (m *MuninRunner) Collect(cache *CredentialCache) (map[string]interface{}, error) {
	command := &exec.Cmd{}

	stdoutPipe, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}

	command.SysProcAttr = &syscall.SysProcAttr{}
	command.SysProcAttr.Credential = &syscall.Credential{}
	if err := m.setCredentials(command, cache); err != nil {
		return nil, err
	}

	command.Path = path.Join(m.scriptPath, m.name)

	m.applyEnv(command, "MUNIN_MASTER_IP", "-")
	m.applyEnv(command, "MUNIN_CAP_MULTIGRAPH", "1")
	m.applyEnv(command, "MUNIN_LIBDIR", "/usr/share/munin")
	m.applyAdditionalEnv(command)
	m.applyPluginStateEnv(command)

	log.Debug("ENV: %v", command.Env)
	log.Debug("Credential: %v", command.SysProcAttr.Credential)

	if err := command.Start(); err != nil {
		return nil, err
	}

	m.outputChan = make(chan interface{}, 1)
	m.waitGroup.Add(2)

	go m.stdoutReader(stdoutPipe)
	go m.processHandler(command)

	m.waitGroup.Wait()

	return m.processOutput()
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

func (m *MuninRunner) stdoutReader(stdoutPipe io.ReadCloser) {
	defer func() {
		m.waitGroup.Done()
	}()

	output, err := ioutil.ReadAll(stdoutPipe)
	if err != nil {
		m.outputChan <- err
		return
	}

	m.outputChan <- output
}

func (m *MuninRunner) processHandler(command *exec.Cmd) {
	defer func() {
		m.waitGroup.Done()
	}()

	var err error

	// Reset the timer
	sawTimeout := false
	m.commandTimer.Reset(10 * time.Second)

	select {
	case output := <-m.outputChan:
		switch v := output.(type) {
		case error:
			err = v
		case []byte:
			m.output = v
		}
	case <-m.commandTimer.C:
		err = ErrCommandTimeout
		sawTimeout = true
		break
	}

	// Stop and clear any stale trigger
	if !m.commandTimer.Stop() && !sawTimeout {
		<-m.commandTimer.C
	}

	if err != nil {
		log.Errorf("Error: %s", err)
		command.Process.Kill()
	}

	command.Wait()
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

func (m *MuninRunner) processOutput() (map[string]interface{}, error) {
	defer func() {
		m.output = nil
	}()

	reader := bytes.NewReader(m.output)
	scanner := bufio.NewScanner(reader)

	noMultigraph := ""
	multigraph := &noMultigraph
	result := map[string]interface{}{}
	for scanner.Scan() {
		if err := m.processOutputLine(scanner.Bytes(), multigraph, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (m *MuninRunner) processOutputLine(line []byte, multigraph *string, result map[string]interface{}) error {
	log.Debug("LINE: %s", line)
	if len(line) == 0 {
		return nil
	}

	line = bytes.Trim(line, " \t")
	if len(line) == 0 {
		return nil
	}

	if line[0] == '#' {
		return nil
	}

	split := bytes.IndexAny(line, " \t")
	if split == -1 {
		return ErrInvalidOutput
	}

	name := string(line[:split])
	value := bytes.TrimLeft(line[split+1:], " \t")
	if name == "multigraph" {
		*multigraph = string(value)
		return nil
	}

	nameSplit := strings.IndexAny(name, ".")
	if nameSplit == -1 {
		return ErrInvalidOutput
	}

	valueType := name[nameSplit+1:]
	if valueType != "value" {
		// Ignore anything but .value entries
		return nil
	}

	floatValue, err := strconv.ParseFloat(string(value), 64)
	if err != nil {
		return err
	}

	name = name[:nameSplit]

	if *multigraph == "" {
		result[name] = floatValue
	} else {
		result[*multigraph+"."+name] = floatValue
	}

	return nil
}
