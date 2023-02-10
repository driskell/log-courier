/*
 * Copyright 2012-2023 Jason Woods and contributors
 *
 * This file is a modification of code from Logstash Forwarder.
 * Copyright 2012-2013 Jordan Sissel and contributors.
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

package transports

import (
	"fmt"

	"github.com/driskell/log-courier/lc-lib/config"
)

type ClientTlsConfiguration struct {
	SSLCA string `config:"ssl ca"`

	*TlsConfiguration `config:",embed"`
}

func (f *ClientTlsConfiguration) TlsValidate(enableTls bool, p *config.Parser, configPath string) (err error) {
	if enableTls {
		if len(f.SSLCA) == 0 {
			return fmt.Errorf("%sssl ca is required for secure transports", configPath)
		}
		if f.CaList, err = AddCertificates(f.CaList, f.SSLCA); err != nil {
			return fmt.Errorf("failure loading %sssl ca: %s", configPath, err)
		}
	} else {
		if len(f.SSLCA) > 0 {
			return fmt.Errorf("%[1]sssl ca is not supported for non-secure transports", configPath)
		}
	}

	return f.TlsConfiguration.TlsValidate(enableTls, p, configPath)
}

func (f *ClientTlsConfiguration) Defaults() {
}

func (f *ClientTlsConfiguration) HasChanged(newConfig *ClientTlsConfiguration) bool {
	if newConfig.SSLCA != f.SSLCA {
		return true
	}
	return f.TlsConfiguration.HasChanged(newConfig.TlsConfiguration)
}
