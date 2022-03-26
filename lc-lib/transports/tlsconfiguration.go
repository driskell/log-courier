/*
 * Copyright 2012-2020 Jason Woods and contributors
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
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"reflect"

	"github.com/driskell/log-courier/lc-lib/config"
)

const (
	// Default to TLS 1.2 minimum, supported since Go 1.2
	defaultMinTLSVersion = tls.VersionTLS12
	defaultMaxTLSVersion = 0
)

type TlsConfiguration struct {
	SSLCertificate      string `config:"ssl certificate"`
	SSLKey              string `config:"ssl key"`
	MinTLSVersionString string `config:"min tls version"`
	MaxTLSVersionString string `config:"max tls version"`

	Certificate     *tls.Certificate
	CertificateList []*x509.Certificate
	CaList          []*x509.Certificate
	MinTLSVersion   uint16
	MaxTLSVersion   uint16
}

func (f *TlsConfiguration) TlsValidate(enableTls bool, p *config.Parser, configPath string) (err error) {
	// Only allow SSL configurations if using TLS
	if !enableTls {
		if len(f.MinTLSVersionString) > 0 || len(f.MaxTLSVersionString) > 0 {
			return fmt.Errorf("%[1]smin tls version and %[1]smax tls version are not supported when the transport is tcp", configPath)
		}
		if len(f.SSLCertificate) > 0 || len(f.SSLKey) > 0 {
			return fmt.Errorf("%[1]sssl certificate and %[1]sssl key are not supported when the transport is tcp", configPath)
		}
		return nil
	}

	// Check tls versions
	f.MinTLSVersion, err = ParseTLSVersion(f.MinTLSVersionString, defaultMinTLSVersion)
	if err != nil {
		return err
	}
	f.MaxTLSVersion, err = ParseTLSVersion(f.MaxTLSVersionString, defaultMaxTLSVersion)
	if err != nil {
		return err
	}

	if len(f.SSLCertificate) > 0 || len(f.SSLKey) > 0 {
		if len(f.SSLCertificate) == 0 {
			return fmt.Errorf("%[1]sssl key is only valid with a matching %[1]sssl certificate", configPath)
		}

		if len(f.SSLKey) == 0 {
			return fmt.Errorf("%[1]sssl key must be specified when %[1]sssl certificate is specified", configPath)
		}

		certificate, err := tls.LoadX509KeyPair(f.SSLCertificate, f.SSLKey)
		if err != nil {
			return fmt.Errorf("failed loading %sssl certificate': %s", configPath, err)
		}

		f.Certificate = &certificate

		for _, certBytes := range f.Certificate.Certificate {
			thisCert, err := x509.ParseCertificate(certBytes)
			if err != nil {
				return fmt.Errorf("failed loading %sssl certificate: %s", configPath, err)
			}
			f.CertificateList = append(f.CertificateList, thisCert)
		}
	}

	return nil
}

func (f *TlsConfiguration) HasChanged(newConfig *TlsConfiguration) bool {
	if newConfig.MinTLSVersion != f.MinTLSVersion {
		return true
	}
	if newConfig.MaxTLSVersion != f.MaxTLSVersion {
		return true
	}
	if newConfig.SSLCertificate != f.SSLCertificate {
		return true
	}
	if newConfig.SSLKey != f.SSLKey {
		return true
	}
	if newConfig.SSLKey != f.SSLKey {
		return true
	}
	if newConfig.Certificate != nil && f.Certificate != nil {
		if !reflect.DeepEqual(newConfig.Certificate.Certificate, f.Certificate.Certificate) {
			return true
		}
	} else if newConfig.Certificate != nil {
		return true
	} else if f.Certificate != nil {
		return true
	}
	if len(newConfig.CaList) != len(f.CaList) {
		return true
	}
	for index := range newConfig.CaList {
		if !reflect.DeepEqual(newConfig.CaList[index].Raw, f.CaList[index].Raw) {
			return true
		}
	}
	return false
}
