package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
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
	scriptPath   string
	name         string
	user         string
	group        string
	command      exec.Cmd
	commandTimer *time.Timer
	stdoutPipe   io.ReadCloser
	outputChan   chan interface{}
	waitGroup    sync.WaitGroup
	output       []byte
}

func NewMuninRunner(scriptPath, name string) *MuninRunner {
	m := &MuninRunner{
		scriptPath:   scriptPath,
		name:         name,
		user:         "nobody",
		group:        "nobody",
		commandTimer: time.NewTimer(time.Second),
	}

	m.command.Path = path.Join(scriptPath, name)
	m.command.SysProcAttr = &syscall.SysProcAttr{}
	m.command.SysProcAttr.Credential = &syscall.Credential{}
	m.applyEnv("MUNIN_MASTER_IP", "-")
	m.applyEnv("MUNIN_CAP_MULTIGRAPH", "1")
	m.applyPluginStateEnv()
	m.commandTimer.Stop()

	return m
}

func (m *MuninRunner) ApplySection(section map[string]string) error {
	for k, v := range section {
		if k == "user" {
			m.user = v
			m.applyPluginStateEnv()
			continue
		}

		if k == "group" {
			m.group = v
			continue
		}

		if strings.HasPrefix(k, environmentConfigPrefix) {
			m.applyEnv(k[len(environmentConfigPrefix):], v)
			continue
		}

		// Ignore
	}

	return nil
}

func (m *MuninRunner) Collect(cache *CredentialCache) (map[string]interface{}, error) {
	var err error

	if m.updateCredentials(cache); err != nil {
		return nil, err
	}

	m.stdoutPipe, err = m.command.StdoutPipe()
	if err != nil {
		return nil, err
	}

	fmt.Printf("ENV: %v\n", m.command.Env)
	fmt.Printf("Credential: %v\n", m.command.SysProcAttr.Credential)

	if err = m.command.Start(); err != nil {
		return nil, err
	}

	m.commandTimer.Reset(10 * time.Second)
	m.outputChan = make(chan interface{}, 1)
	m.waitGroup.Add(2)

	go m.stdoutReader()
	go m.processHandler()

	m.waitGroup.Wait()

	return m.processOutput()
}

func (m *MuninRunner) updateCredentials(cache *CredentialCache) error {
	if err := m.applyCommandUser(m.user, cache); err != nil {
		return err
	}

	if err := m.applyCommandGroup(m.group, cache); err != nil {
		return err
	}

	return nil
}

func (m *MuninRunner) stdoutReader() {
	defer func() {
		m.waitGroup.Done()
	}()

	output, err := ioutil.ReadAll(m.stdoutPipe)
	if err != nil {
		m.outputChan <- err
		return
	}

	m.outputChan <- output
}

func (m *MuninRunner) processHandler() {
	defer func() {
		m.waitGroup.Done()
	}()

	var err error

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
		break
	}

	if err != nil {
		fmt.Printf("Error: %s\n", err)
		m.command.Process.Kill()
	}

	m.command.Wait()
}

func (m *MuninRunner) applyEnv(name string, value string) {
	// Validate. This needs checking
	// I'm guessing here from some things I read about IEEE Std 1003.1-2001
	name = strings.Replace(name, "\x00", "", -1)
	name = strings.Replace(name, "=", "", -1)

	m.command.Env = append(m.command.Env, name+"="+value)
}

func (m *MuninRunner) applyPluginStateEnv() {
	m.applyEnv("MUNIN_PLUGSTATE", "/var/lib/munin/plugin-state/"+m.user)
	m.applyEnv("MUNIN_STATEFILE", "/var/lib/munin/plugin-state/"+m.user+"/"+m.name+"-")
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
	fmt.Printf("LINE: %s\n", line)
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
