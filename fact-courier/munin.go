package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/publisher"
)

const (
	collectionInterval = time.Minute
)

// MuninCollector facilitates the collection of statistics from a local
// munin-node installation
type MuninCollector struct {
	core.PipelineSegment

	app        *core.App
	factConfig *Config
	genConfig  *config.General

	muninConfig *MuninConfig
	sections    []string
	runners     []*MuninRunner

	credentialCache *CredentialCache
	publisher       *publisher.Publisher
	output          chan<- []*event.Event
}

// NewMuninCollector creates a new MuninCollector for the given app with the
// given publisher as the output for events
func NewMuninCollector(app *core.App, publisher *publisher.Publisher) (*MuninCollector, error) {
	ret := &MuninCollector{
		app:             app,
		factConfig:      app.Config().Section("facts").(*Config),
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

// init loads the munin-node configuration and detects the plugins to use, and
// creates a MuninRunner for each one
func (m *MuninCollector) init() error {
	if err := m.loadConfig(); err != nil {
		return err
	}

	m.loadScripts()

	return nil
}

// Run the MuninCollector - loops until shutdown
func (m *MuninCollector) Run() {
	defer func() {
		m.Done()
	}()

	for m.runOnce() {
	}

	log.Info("Munin collector exiting")
}

// runOnce performs a single collection of data
// It will wait until the next collectionInterval boundary (such as 1 minute
// boundary) so that all collections happen at the same boundary regardless of
// when Fact Courier was started
// This also ensures that every collection result has a single round timestamp
// to ensure that Kibana graphing is neat with no gaps
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

	// TODO: What if too many events? Bundle could get too big...
	events := m.collect(nextCollection)
	if len(events) != 0 {
		if !m.sendEvents(events) {
			return false
		}
	}

	return true
}

// collect performs a collection from all discovered munin plugins by collecting
// from each MuninRunner we created
func (m *MuninCollector) collect(timestamp time.Time) []*event.Event {
	events := make([]*event.Event, 0, len(m.runners))
	for _, runner := range m.runners {
		result := m.collectRunner(runner, timestamp)
		if result != nil {
			events = append(events, result)
		}
	}
	return events
}

// collectRunner performs a collection from a single MuninRunner, and generates
// the necessary event structure from the result
func (m *MuninCollector) collectRunner(runner *MuninRunner, timestamp time.Time) *event.Event {
	log.Debug("[%s] Collecting", runner.Name())

	result, err := runner.Collect(m.credentialCache, timestamp)
	if err != nil {
		log.Warning("[%s] Collect error: %s", runner.Name(), err)
		return nil
	}

	// Set timestamp, plugin name and other fields
	result["@timestamp"] = timestamp.Format(time.RFC3339)
	result["munin_plugin"] = runner.Name()

	// Create a new munin event with nil context
	event := m.factConfig.NewEvent("munin", result, nil)

	// Pre-encode it for transmission
	err = event.Encode()
	if err != nil {
		log.Warning("[%s] Skipping data due to encoding error: %s", runner.Name(), err)
		return nil
	}

	return event
}

// sendEvents passes a bundle of events to the publisher, waiting on shutdown
// as well in case of a slow pipeline
func (m *MuninCollector) sendEvents(events []*event.Event) bool {
	select {
	case m.output <- events:
		return true
	case <-m.OnShutdown():
	}
	return false
}

// scanFolder scans the requested folder and calls the callback for each file
// within it
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

// loadConfig loads the requested munin-node configuration for parsing, it also
// scans the plugin-conf.d folder for plugin configurations so we can ensure
// we run plugins as the correct user and group
func (m *MuninCollector) loadConfig() error {
	// TODO: Configuration for this
	muninConfig, err := NewMuninConfig("/etc/munin/munin-node.conf")
	if err != nil {
		return err
	}

	// TODO: Determine this from the munin-node.conf? Can it be changed there?
	confPath := "/etc/munin/plugin-conf.d"
	m.scanFolder(confPath, func(name string, err error) {
		if err != nil {
			log.Errorf("Config scan error: %s", err)
			return
		}

		err = muninConfig.Append(filepath.Join(confPath, name))
		if err != nil {
			log.Errorf("Config append error: %s", err)
			return
		}
	})

	m.muninConfig = muninConfig
	m.sections = m.muninConfig.Sections()

	log.Debug("Sections: %v", m.sections)
	for _, section := range m.sections {
		log.Debug("[%s]: %v", section, m.muninConfig.Section(section))
	}

	return nil
}

// loadScripts performs munin-node plugin discovery
func (m *MuninCollector) loadScripts() {
	// TODO: Determine this from the munin-node.conf? Can it be changed there?
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

// createRunner scans the loaded plugin configurations to produce a list of
// those that need to be provided to the MuninRunner so that it can update its
// user/group and environment configurations
// It then passes these to createRunnerFromSections and returns the MuninRunner
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

// createRunnerFromSections creates a MuninRunner from the given set of plugin
// configurations
func (m *MuninCollector) createRunnerFromSections(scriptPath, name string, sections []string) (*MuninRunner, error) {
	runner := NewMuninRunner(scriptPath, name)

	for _, section := range sections {
		log.Debug("Applying %s to %s", section, name)
		if err := runner.ApplySection(m.muninConfig.Section(section)); err != nil {
			return nil, err
		}
	}

	log.Debug("Configuring %s", name)
	if err := runner.Configure(m.credentialCache); err != nil {
		return nil, err
	}

	return runner, nil
}
