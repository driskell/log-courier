package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type MuninCollector struct {
	config   *MuninConfig
	sections []string
	runners  []*MuninRunner
}

func NewMuninCollector() (*MuninCollector, error) {
	ret := &MuninCollector{}

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

func (m *MuninCollector) Collect() {
	cache := NewCredentialCache()

	for _, runner := range m.runners {
		fmt.Printf("RUNNING: %s\n", runner.name)
		m.collectRunner(runner, cache)
	}
}

func (m *MuninCollector) collectRunner(runner *MuninRunner, cache *CredentialCache) {
	result, err := runner.Collect(cache)
	if err != nil {
		fmt.Printf("Collect error: %s\n", err)
		return
	}

	fmt.Printf("SUCCESS: %s\n", result)
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
			fmt.Printf("Config scan error: %s\n", err)
			return
		}

		err = config.Append(filepath.Join(confPath, name))
		if err != nil {
			fmt.Printf("Config append error: %s\n", err)
			return
		}
	})

	m.config = config
	m.sections = m.config.Sections()

	fmt.Printf("Sections: %v\n", m.sections)
	for _, section := range m.sections {
		fmt.Printf("[%s]: %v\n", section, m.config.Section(section))
	}

	return nil
}

func (m *MuninCollector) loadScripts() {
	scriptPath := "/etc/munin/plugins"
	m.scanFolder(scriptPath, func(name string, err error) {
		if err != nil {
			fmt.Printf("Scan error: %s\n", err)
			return
		}

		runner, err := m.createRunner(scriptPath, name)
		if err != nil {
			fmt.Printf("Create runner error: %s\n", err)
			return
		}

		m.runners = append(m.runners, runner)
	})
}

func (m *MuninCollector) createRunner(scriptPath, name string) (*MuninRunner, error) {
	var applySections []string

	fmt.Printf("Creating runner %s\n", name)

	// Search sections and match against name
	for _, section := range m.sections {
		fmt.Printf("Considering %s\n", section)
		sectionLen := len(section)
		if section[sectionLen-1] == '*' {
			if !strings.HasPrefix(name, section[:sectionLen-1]) {
				continue
			}
		} else if section != name {
			continue
		}

		fmt.Printf("Using %s\n", section)

		applySections = append(applySections, section)
	}

	return m.createRunnerFromSections(scriptPath, name, applySections)
}

func (m *MuninCollector) createRunnerFromSections(scriptPath, name string, sections []string) (*MuninRunner, error) {
	runner := NewMuninRunner(scriptPath, name)
	for _, section := range sections {
		fmt.Printf("Applying %s to %s\n", section, name)
		if err := runner.ApplySection(m.config.Section(section)); err != nil {
			return nil, err
		}
	}

	return runner, nil
}
