package geoipupdate

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/maxmind/geoipupdate/v4/pkg/geoipupdate"
	"github.com/maxmind/geoipupdate/v4/pkg/geoipupdate/database"
)

var (
	editionIDs = []string{
		"GeoLite2-City",
	}
)

// General contains general configuration values
type General struct {
	AccountId  int    `config:"geoip account id"`
	LicenseKey string `config:"geoip license key"`
}

// GetDatabasePath returns the path to the GeoIP database file
func GetDatabasePath(config *config.Config) string {
	return filepath.Join(config.General().PersistDir, "GeoLite2-City.mmdb")
}

// StartUpdater starts the GeoIP database updater routine
func StartUpdater(config *config.Config) (chan<- struct{}, *sync.WaitGroup) {
	shutdown := make(chan struct{})
	waitGroup := &sync.WaitGroup{}

	generalConfig := config.GeneralPart("geoipupdate").(*General)

	updateConfig := &geoipupdate.Config{
		AccountID:         generalConfig.AccountId,
		DatabaseDirectory: config.General().PersistDir,
		LicenseKey:        generalConfig.LicenseKey,
		LockFile:          filepath.Join(config.General().PersistDir, ".geoipupdate.lock"),
		URL:               "https://updates.maxmind.com",
		EditionIDs:        editionIDs,
		Verbose:           false,
	}

	if err := checkAndUpdate(updateConfig); err != nil {
		log.Errorf("Failed to update GeoIP database: %s", err)
	}

	waitGroup.Add(1)
	go updaterRoutine(updateConfig, shutdown, waitGroup)

	return shutdown, waitGroup
}

// updaterRoutine runs the GeoIP database update routine
func updaterRoutine(config *geoipupdate.Config, shutdown <-chan struct{}, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	for {
		select {
		case <-shutdown:
			return
		case <-time.After(time.Hour*24 + time.Minute*time.Duration(rand.Intn(60))):
		}

		if err := checkAndUpdate(config); err != nil {
			log.Errorf("Failed to update GeoIP database: %s", err)
		}
	}
}

// checkAndUpdate checks for and performs GeoIP database updates
func checkAndUpdate(config *geoipupdate.Config) error {
	checkFile := filepath.Join(config.DatabaseDirectory, ".geoipupdate-check")
	if fileInfo, err := os.Stat(checkFile); !os.IsNotExist(err) {
		if time.Since(fileInfo.ModTime()) < time.Hour*24 {
			return nil
		}
	}

	log.Infof("Checking for GeoIP database updates")

	client := geoipupdate.NewClient(config)
	dbReader := database.NewHTTPDatabaseReader(client, config)

	for _, editionID := range config.EditionIDs {
		filename, err := geoipupdate.GetFilename(config, editionID, client)
		if err != nil {
			return fmt.Errorf("error retrieving filename for %s: %w", editionID, err)
		}
		filePath := filepath.Join(config.DatabaseDirectory, filename)
		dbWriter, err := database.NewLocalFileDatabaseWriter(filePath, config.LockFile, config.Verbose)
		if err != nil {
			return fmt.Errorf("error creating database writer for %s: %w", editionID, err)
		}
		if err := dbReader.Get(dbWriter, editionID); err != nil {
			return fmt.Errorf("error while getting database for %s: %w", editionID, err)
		}
		log.Noticef("Successfully downloaded GeoIP database: %s", filePath)
	}

	now := time.Now()
	if _, err := os.Stat(checkFile); os.IsNotExist(err) {
		emptyFile, err := os.Create(checkFile)
		if err != nil {
			return fmt.Errorf("error creating check file: %w", err)
		}
		emptyFile.Close()
	} else if err := os.Chtimes(checkFile, now, now); err != nil {
		return fmt.Errorf("error updating check file time: %w", err)
	}

	log.Infof("GeoIP database update check complete")

	return nil
}

// init registers this module provider
func init() {
	config.RegisterGeneral("geoipupdate", func() interface{} {
		return &General{}
	})
}
