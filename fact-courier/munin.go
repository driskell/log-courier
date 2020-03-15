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
)

const (
	collectionInterval = time.Minute
)

// MuninCollector facilitates the collection of statistics from a local
// munin-node installation
type MuninCollector struct {
	factConfig *Config
	genConfig  *config.General

	muninConfig *MuninConfig
	sections    []string
	runners     []*MuninRunner

	credentialCache *CredentialCache
	shutdownChan    <-chan struct{}
	output          chan<- []*event.Event
}

// NewMuninCollector creates a new MuninCollector for the given app with the
// given publisher as the output for events
func NewMuninCollector(app *core.App) *MuninCollector {
	return &MuninCollector{
		factConfig:      app.Config().Section("facts").(*Config),
		genConfig:       app.Config().General(),
		credentialCache: NewCredentialCache(),
	}
}

// Init loads the munin-node configuration and detects the plugins to use, and
// creates a MuninRunner for each one, returning an error if any issues
func (m *MuninCollector) Init(cfg *config.Config) error {
	if err := m.loadConfig(); err != nil {
		return err
	}

	m.loadScripts()
	return nil
}

// SetOutput sets the output channel
func (m *MuninCollector) SetOutput(output chan<- []*event.Event) {
	m.output = output
}

// SetShutdownChan sets the shutdown channel
func (m *MuninCollector) SetShutdownChan(shutdownChan <-chan struct{}) {
	m.shutdownChan = shutdownChan
}

// Run the MuninCollector - loops until shutdown
func (m *MuninCollector) Run() {
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
	case <-m.shutdownChan:
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
	result, err := runner.Collect(m.credentialCache, timestamp)
	if err != nil {
		log.Warning("[%s] Collect error: %s", runner.Name(), err)
		return nil
	}

	fieldCount := len(result)

	// Set timestamp, plugin name and other fields
	result["@timestamp"] = timestamp.Format(time.RFC3339)
	result["munin_plugin"] = runner.Name()

	// Create a new munin event with nil context
	result = m.factConfig.Decorate(result)
	event := event.NewEvent(nil, nil, result)

	log.Debug("[%s] %d fields collected", runner.Name(), fieldCount)

	return event
}

// sendEvents passes a bundle of events to the publisher, waiting on shutdown
// as well in case of a slow pipeline
func (m *MuninCollector) sendEvents(events []*event.Event) bool {
	select {
	case m.output <- events:
		return true
	case <-m.shutdownChan:
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
	log.Debug("Loading munin configuration from: %s", m.factConfig.MuninConfigFile)
	muninConfig, err := NewMuninConfig(m.factConfig.MuninConfigFile)
	if err != nil {
		return err
	}

	m.scanFolder(m.factConfig.MuninConfigPluginD, func(name string, err error) {
		if err != nil {
			log.Errorf("Config scan error: %s", err)
			return
		}

		confPath := filepath.Join(m.factConfig.MuninConfigPluginD, name)

		log.Debug("Loading additional munin configuration from: %s", confPath)
		err = muninConfig.Append(confPath)
		if err != nil {
			log.Errorf("Config append error: %s", err)
			return
		}
	})

	m.muninConfig = muninConfig
	m.sections = m.muninConfig.Sections()

	for _, section := range m.sections {
		log.Debug("[%s]: %v", section, m.muninConfig.Section(section))
	}

	log.Info("Successfully loaded %d munin configuration sections", len(m.sections))

	return nil
}

// loadScripts performs munin-node plugin discovery
func (m *MuninCollector) loadScripts() {
	m.scanFolder(m.factConfig.MuninPluginBase, func(name string, err error) {
		if err != nil {
			log.Errorf("Scan error: %s", err)
			return
		}

		runner, err := m.createRunner(m.factConfig.MuninPluginBase, name)
		if err != nil {
			log.Errorf("Create runner error: %s", err)
			return
		}

		m.runners = append(m.runners, runner)
	})

	log.Info("Successfully configured %d munin plugin runners", len(m.runners))
}

// createRunner scans the loaded plugin configurations to produce a list of
// those that need to be provided to the MuninRunner so that it can update its
// user/group and environment configurations
// It then passes these to createRunnerFromSections and returns the MuninRunner
func (m *MuninCollector) createRunner(scriptPath, name string) (*MuninRunner, error) {
	var applySections []string

	log.Debug("[%s] Configuring munin plugin runner", name)

	// Search sections and match against name
	for _, section := range m.sections {
		sectionLen := len(section)
		if section[sectionLen-1] == '*' {
			if !strings.HasPrefix(name, section[:sectionLen-1]) {
				continue
			}
		} else if section != name {
			continue
		}

		log.Debug("[%s] Using configuration section: %s", name, section)

		applySections = append(applySections, section)
	}

	return m.createRunnerFromSections(scriptPath, name, applySections)
}

// createRunnerFromSections creates a MuninRunner from the given set of plugin
// configurations
func (m *MuninCollector) createRunnerFromSections(scriptPath, name string, sections []string) (*MuninRunner, error) {
	runner := NewMuninRunner(scriptPath, name)

	for _, section := range sections {
		if err := runner.ApplySection(m.muninConfig.Section(section)); err != nil {
			return nil, err
		}
	}

	if err := runner.Configure(m.credentialCache); err != nil {
		return nil, err
	}

	return runner, nil
}
