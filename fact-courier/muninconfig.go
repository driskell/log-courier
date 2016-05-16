package main

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
)

var (
	globalSection = "global"

	noActiveContinuation = ""

	// ErrInvalidSection is reported when a corrupted section line is encountered
	ErrInvalidSection = errors.New("Invalid section")
	// ErrInvalidVariable is reported when a corrupted variable line is encountered
	ErrInvalidVariable = errors.New("Invalid variable")
)

// MuninConfig parses munin-node configurations and provides an interface to
// accessing it
type MuninConfig struct {
	sections map[string]map[string]string
}

// NewMuninConfig creates a new MuninConfig for configuration parsing and parses
// the given configuration file
func NewMuninConfig(path string) (*MuninConfig, error) {
	ret := &MuninConfig{
		sections: map[string]map[string]string{
			"global": map[string]string{},
		},
	}

	if err := ret.Append(path); err != nil {
		return nil, err
	}

	return ret, nil
}

// Append parses another configuration file and appends the configuration to
// that already loaded
// It is used to load plugin-conf.d files
func (m *MuninConfig) Append(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	defer func() {
		file.Close()
	}()

	return m.ingestReader(file)
}

// AddSection adds a new section to the MuninConfig
func (m *MuninConfig) AddSection(name string) map[string]string {
	if section := m.Section(name); section != nil {
		return section
	}

	section := map[string]string{}
	m.sections[name] = section
	return section
}

// Section returns the requested section from the MuninConfig by name
func (m *MuninConfig) Section(name string) map[string]string {
	if section, exists := m.sections[name]; exists {
		return section
	}

	return nil
}

// Sections returns a list of the existing sections within MuninConfig
func (m *MuninConfig) Sections() []string {
	var keys []string
	for k := range m.sections {
		keys = append(keys, k)
	}
	return keys
}

// ingestReader reads the configuration from the given reader and parses it
func (m *MuninConfig) ingestReader(reader io.Reader) error {
	activeSection := &globalSection
	activeContinuation := &noActiveContinuation
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		if err := m.processLine(scanner.Bytes(), activeSection, activeContinuation); err != nil {
			return err
		}
	}

	return nil
}

// processLine takes a single line of a configuration file and parses it
func (m *MuninConfig) processLine(line []byte, activeSection *string, activeContinuation *string) error {
	if len(line) == 0 {
		return nil
	}

	line = bytes.Trim(line, " \t")
	lenLine := len(line)
	if lenLine == 0 {
		return nil
	}

	if line[0] == '#' {
		return nil
	}

	if line[0] == '[' {
		if line[lenLine-1] != ']' {
			return ErrInvalidSection
		}

		sectionName := string(line[1 : lenLine-1])
		m.AddSection(sectionName)
		*activeSection = sectionName
		return nil
	}

	if *activeContinuation != noActiveContinuation {
		if line[lenLine-1] == '\\' {
			line = line[:lenLine-1]
		} else {
			*activeContinuation = ""
		}

		m.sections[*activeSection][*activeContinuation] = m.sections[*activeSection][*activeContinuation] + string(line)
		return nil
	}

	split := bytes.IndexAny(line, " \t")
	if split < 1 {
		return ErrInvalidVariable
	}

	name := string(line[:split])
	line = bytes.TrimLeft(line[split+1:], " \t")
	lenLine = len(line)

	if line[lenLine-1] == '\\' {
		line = line[:lenLine-1]
		*activeContinuation = name
	}

	m.sections[*activeSection][name] = string(line)
	return nil
}
