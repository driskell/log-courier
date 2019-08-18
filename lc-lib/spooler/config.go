/*
 * Copyright 2014-2016 Jason Woods.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package spooler

import (
	"fmt"
	"time"

	"github.com/driskell/log-courier/lc-lib/config"
	"github.com/driskell/log-courier/lc-lib/harvester"
)

const (
	defaultGeneralSpoolMaxBytes int64         = 10485760
	defaultGeneralSpoolSize     int64         = 1024
	defaultGeneralSpoolTimeout  time.Duration = 5 * time.Second
)

// General contains general configuration values
type General struct {
	SpoolSize     int64         `config:"spool size"`
	SpoolMaxBytes int64         `config:"spool max bytes"`
	SpoolTimeout  time.Duration `config:"spool timeout"`
}

// Validate the additional general configuration
func (gc *General) Validate(p *config.Parser, path string) (err error) {
	// Enforce maximum of 2 GB since event transmit length is uint32
	if gc.SpoolMaxBytes > 2*1024*1024*1024 {
		err = fmt.Errorf("%s/spool max bytes can not be greater than 2 GiB", path)
		return
	}

	// Max line bytes can not be larger than spool max bytes
	if gc.SpoolMaxBytes < p.Config().GeneralPart("harvester").(*harvester.General).MaxLineBytes {
		err = fmt.Errorf("%s/max line bytes can not be greater than %s/spool max bytes", path, path)
		return
	}

	return
}

func init() {
	config.RegisterGeneral("spooler", func() interface{} {
		return &General{
			SpoolSize:     defaultGeneralSpoolSize,
			SpoolMaxBytes: defaultGeneralSpoolMaxBytes,
			SpoolTimeout:  defaultGeneralSpoolTimeout,
		}
	})
}
