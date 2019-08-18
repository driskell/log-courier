package stdinharvester

import (
	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/core"
	"github.com/driskell/log-courier/lc-lib/event"
	"github.com/driskell/log-courier/lc-lib/harvester"
)

// Stdin harvests logs from STDIN
type Stdin struct {
	outputChan   chan<- []*event.Event
	shutdownChan <-chan struct{}
	harvester    *harvester.Harvester
}

// New creates new instance
func New(app *core.App) *Stdin {
	cfg := app.Config()
	streamConfig := cfg.Section("stdin").(*StreamConfig)
	return &Stdin{
		harvester: streamConfig.NewHarvester(cfg, nil, nil, 0),
	}
}

// Init does nothing as nothing to init
func (s *Stdin) Init(*config.Config) error {
	return nil
}

// SetOutput sets the output channel
func (s *Stdin) SetOutput(outputChan chan<- []*event.Event) {
	s.outputChan = outputChan
}

// SetShutdownChan sets the shutdown channel
func (s *Stdin) SetShutdownChan(shutdownChan <-chan struct{}) {
	s.shutdownChan = shutdownChan
}

// Run the harvester
func (s *Stdin) Run() {
	s.harvester.SetOutput(s.outputChan)
	s.harvester.Start()

	finished := <-s.harvester.OnFinish()

	if finished.Error != nil {
		log.Notice("An error occurred reading from stdin at offset %d: %s", finished.LastReadOffset, finished.Error)
	} else {
		log.Notice("Finished reading from stdin at offset %d", finished.LastReadOffset)
	}
}
