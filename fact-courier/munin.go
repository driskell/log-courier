package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/publisher"
)

const (
	collectionInterval = time.Minute
)

type MuninCollector struct {
	core.PipelineSegment

	app       *core.App
	genConfig *config.General

	config   *MuninConfig
	sections []string
	runners  []*MuninRunner

	credentialCache *CredentialCache
	publisher       *publisher.Publisher
	output          chan<- []*core.EventDescriptor
}

func NewMuninCollector(app *core.App, publisher *publisher.Publisher) (*MuninCollector, error) {
	ret := &MuninCollector{
		app:             app,
		genConfig:       app.Config().General(),
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

	log.Info("Munin collector exiting")
}

func (m *MuninCollector) runOnce() bool {
	nextCollection := time.Now().Truncate(collectionInterval).Add(collectionInterval)
	delay := nextCollection.Sub(time.Now())
	if delay > collectionInterval {
		delay = collectionInterval
	} else if delay <= 0 {
		delay = 0
	}

	log.Debug("Next collection at %s in %s", nextCollection, delay)

	select {
	case <-m.OnShutdown():
		return false
	case <-time.After(delay):
	}

	events := m.collect(nextCollection)
	if len(events) != 0 {
		if !m.sendEvents(events) {
			return false
		}
	}

	return true
}

func (m *MuninCollector) collect(timestamp time.Time) []*core.EventDescriptor {
	events := make([]*core.EventDescriptor, 0, len(m.runners))
	for _, runner := range m.runners {
		result := m.collectRunner(runner, timestamp)
		if result != nil {
			events = append(events, result)
		}
	}
	return events
}

func (m *MuninCollector) collectRunner(runner *MuninRunner, timestamp time.Time) *core.EventDescriptor {
	log.Debug("[%s] Collecting", runner.Name())

	result, err := runner.Collect(m.credentialCache, timestamp)
	if err != nil {
		log.Warning("[%s] Collect error: %s", runner.Name(), err)
		return nil
	}

	event := core.Event(result)

	event["@timestamp"] = timestamp.Format(time.RFC3339)

	event["plugin"] = runner.Name()

	for k := range m.genConfig.GlobalFields {
		event[k] = m.genConfig.GlobalFields[k]
	}

	encodedEvent, err := event.Encode()
	if err != nil {
		log.Warning("[%s] Skipping data due to encoding error: %s", runner.Name(), err)
		return nil
	}

	return &core.EventDescriptor{
		Stream: nil,
		Offset: 0,
		Event:  encodedEvent,
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
			log.Errorf("Config scan error: %s", err)
			return
		}

		err = config.Append(filepath.Join(confPath, name))
		if err != nil {
			log.Errorf("Config append error: %s", err)
			return
		}
	})

	m.config = config
	m.sections = m.config.Sections()

	log.Debug("Sections: %v", m.sections)
	for _, section := range m.sections {
		log.Debug("[%s]: %v", section, m.config.Section(section))
	}

	return nil
}

func (m *MuninCollector) loadScripts() {
	scriptPath := "/etc/munin/plugins"
	m.scanFolder(scriptPath, func(name string, err error) {
		if err != nil {
			log.Errorf("Scan error: %s", err)
			return
		}

		runner, err := m.createRunner(scriptPath, name)
		if err != nil {
			log.Errorf("Create runner error: %s", err)
			return
		}

		m.runners = append(m.runners, runner)
	})
}

func (m *MuninCollector) createRunner(scriptPath, name string) (*MuninRunner, error) {
	var applySections []string

	log.Debug("Creating runner %s", name)

	// Search sections and match against name
	for _, section := range m.sections {
		log.Debug("Considering %s", section)
		sectionLen := len(section)
		if section[sectionLen-1] == '*' {
			if !strings.HasPrefix(name, section[:sectionLen-1]) {
				continue
			}
		} else if section != name {
			continue
		}

		log.Debug("Using %s", section)

		applySections = append(applySections, section)
	}

	return m.createRunnerFromSections(scriptPath, name, applySections)
}

func (m *MuninCollector) createRunnerFromSections(scriptPath, name string, sections []string) (*MuninRunner, error) {
	runner := NewMuninRunner(scriptPath, name)

	for _, section := range sections {
		log.Debug("Applying %s to %s", section, name)
		if err := runner.ApplySection(m.config.Section(section)); err != nil {
			return nil, err
		}
	}

	log.Debug("Configuring %s", name)
	if err := runner.Configure(m.credentialCache); err != nil {
		return nil, err
	}

	return runner, nil
}
