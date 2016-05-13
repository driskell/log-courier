package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/publisher"
)

type MuninCollector struct {
	core.PipelineSegment

	config   *MuninConfig
	sections []string
	runners  []*MuninRunner

	credentialCache *CredentialCache
	publisher       *publisher.Publisher
	output          chan<- []*core.EventDescriptor
}

func NewMuninCollector(publisher *publisher.Publisher) (*MuninCollector, error) {
	ret := &MuninCollector{
		publisher:       publisher,
		output:          publisher.Connect(),
		credentialCache: NewCredentialCache(),
	}

	if err := ret.init(); err != nil {
		return nil, err
	}

	return ret, nil
}

func (m *MuninCollector) init() error {
	if err := m.loadConfig(); err != nil {
		return err
	}

	m.loadScripts()

	return nil
}

func (m *MuninCollector) Run() {
	defer func() {
		m.Done()
	}()

	for m.runOnce() {
	}
}

func (m *MuninCollector) runOnce() bool {
	if !m.sendEvents(m.collect()) {
		return false
	}

	select {
	case <-m.OnShutdown():
		return false
	case <-time.After(60 * time.Second):
	}

	return true
}

func (m *MuninCollector) collect() []*core.EventDescriptor {
	events := make([]*core.EventDescriptor, 0, len(m.runners))
	for _, runner := range m.runners {
		result := m.collectRunner(runner)
		if result != nil {
			events = append(events, result)
		}
	}
	return events
}

func (m *MuninCollector) collectRunner(runner *MuninRunner) *core.EventDescriptor {
	log.Debug("[%s] Collecting\n", runner.Name())

	result, err := runner.Collect(m.credentialCache)
	if err != nil {
		log.Warning("[%s] Collect error: %s\n", runner.Name(), err)
		return nil
	}

	event, err := core.Event(result).Encode()
	if err != nil {
		log.Warning("[%s] Skipping data due to encoding error: %s", runner.Name(), err)
		return nil
	}

	return &core.EventDescriptor{
		Stream: nil,
		Offset: 0,
		Event:  event,
	}
}

func (m *MuninCollector) sendEvents(events []*core.EventDescriptor) bool {
	select {
	case m.output <- events:
		return true
	case <-m.OnShutdown():
	}
	return false
}

func (m *MuninCollector) scanFolder(path string, cb func(string, error)) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}

	finished := false
	for !finished {
		names, err := dir.Readdirnames(32)

		if len(names) == 0 {
			finished = true
		} else {
			for _, name := range names {
				cb(name, nil)
			}
		}

		if err != nil && err != io.EOF {
			cb("", err)
		}
	}

	return nil
}

func (m *MuninCollector) loadConfig() error {
	config, err := NewMuninConfig("/etc/munin/munin-node.conf")
	if err != nil {
		return err
	}

	confPath := "/etc/munin/plugin-conf.d"
	m.scanFolder(confPath, func(name string, err error) {
		if err != nil {
			log.Errorf("Config scan error: %s\n", err)
			return
		}

		err = config.Append(filepath.Join(confPath, name))
		if err != nil {
			log.Errorf("Config append error: %s\n", err)
			return
		}
	})

	m.config = config
	m.sections = m.config.Sections()

	log.Debug("Sections: %v\n", m.sections)
	for _, section := range m.sections {
		log.Debug("[%s]: %v\n", section, m.config.Section(section))
	}

	return nil
}

func (m *MuninCollector) loadScripts() {
	scriptPath := "/etc/munin/plugins"
	m.scanFolder(scriptPath, func(name string, err error) {
		if err != nil {
			log.Errorf("Scan error: %s\n", err)
			return
		}

		runner, err := m.createRunner(scriptPath, name)
		if err != nil {
			log.Errorf("Create runner error: %s\n", err)
			return
		}

		m.runners = append(m.runners, runner)
	})
}

func (m *MuninCollector) createRunner(scriptPath, name string) (*MuninRunner, error) {
	var applySections []string

	log.Debug("Creating runner %s\n", name)

	// Search sections and match against name
	for _, section := range m.sections {
		log.Debug("Considering %s\n", section)
		sectionLen := len(section)
		if section[sectionLen-1] == '*' {
			if !strings.HasPrefix(name, section[:sectionLen-1]) {
				continue
			}
		} else if section != name {
			continue
		}

		log.Debug("Using %s\n", section)

		applySections = append(applySections, section)
	}

	return m.createRunnerFromSections(scriptPath, name, applySections)
}

func (m *MuninCollector) createRunnerFromSections(scriptPath, name string, sections []string) (*MuninRunner, error) {
	runner := NewMuninRunner(scriptPath, name)
	for _, section := range sections {
		log.Debug("Applying %s to %s\n", section, name)
		if err := runner.ApplySection(m.config.Section(section)); err != nil {
			return nil, err
		}
	}

	return runner, nil
}
