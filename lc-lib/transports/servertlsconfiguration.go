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
	"reflect"

	"github.com/driskell/log-courier/lc-lib/config"
)

const (
	defaultSSLVerifyPeers = true
)

type ServerTlsConfiguration struct {
	SSLClientCA    []string `config:"ssl client ca"`
	SSLVerifyPeers bool     `config:"verify peers"`

	*TlsConfiguration `config:",embed"`
}

func (f *ServerTlsConfiguration) TlsValidate(enableTls bool, p *config.Parser, configPath string) (err error) {
	if enableTls {
		if len(f.SSLCertificate) == 0 || len(f.SSLKey) == 0 {
			return fmt.Errorf("%sssl certificate and ssl key must be specified for secure receivers", configPath)
		}

		for idx, clientCA := range f.SSLClientCA {
			if f.CaList, err = AddCertificates(f.CaList, clientCA); err != nil {
				return fmt.Errorf("failure loading %sssl client ca[%d]: %s", configPath, idx, err)
			}
		}
	} else {
		if len(f.SSLClientCA) > 0 {
			return fmt.Errorf("%sssl client ca is not supported for non-secure receivers", configPath)
		}
	}

	return f.TlsConfiguration.TlsValidate(enableTls, p, configPath)
}

func (f *ServerTlsConfiguration) Defaults() {
	f.SSLVerifyPeers = defaultSSLVerifyPeers
}

func (f *ServerTlsConfiguration) HasChanged(newConfig *ServerTlsConfiguration) bool {
	if !reflect.DeepEqual(newConfig.SSLClientCA, f.SSLClientCA) {
		return true
	}
	return f.TlsConfiguration.HasChanged(newConfig.TlsConfiguration)
}
