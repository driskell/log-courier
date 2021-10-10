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

package tcp

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"

	"github.com/driskell/log-courier/lc-lib/config"
)

type TlsConfiguration struct {
	SSLCertificate string `config:"ssl certificate"`
	SSLKey         string `config:"ssl key"`
	MinTLSVersion  string `config:"min tls version"`
	MaxTLSVersion  string `config:"max tls version"`

	certificate     *tls.Certificate
	certificateList []*x509.Certificate
	caList          []*x509.Certificate
	minTLSVersion   uint16
	maxTLSVersion   uint16
}

func (f *TlsConfiguration) tlsValidate(transport string, p *config.Parser, configPath string) (err error) {
	// Check tls versions
	f.minTLSVersion, err = parseTLSVersion(f.MinTLSVersion, defaultMinTLSVersion)
	if err != nil {
		return err
	}
	f.maxTLSVersion, err = parseTLSVersion(f.MaxTLSVersion, defaultMaxTLSVersion)
	if err != nil {
		return err
	}

	// Only allow SSL configurations if using TLS
	if transport != TransportTCPTLS {
		if len(f.SSLCertificate) > 0 || len(f.SSLKey) > 0 {
			return fmt.Errorf("%[1]sssl certificate and %[1]sssl key are not supported when the transport is tcp", configPath)
		}
		return nil
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

		f.certificate = &certificate

		for _, certBytes := range f.certificate.Certificate {
			thisCert, err := x509.ParseCertificate(certBytes)
			if err != nil {
				return fmt.Errorf("failed loading %sssl certificate: %s", configPath, err)
			}
			f.certificateList = append(f.certificateList, thisCert)
		}
	}

	return nil
}

func (f *TlsConfiguration) addCa(file string, configPath string) error {
	pemdata, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed loading %s: %s", configPath, err)
	}
	rest := pemdata
	var block *pem.Block
	var pemBlockNum = 1
	for {
		block, rest = pem.Decode(rest)
		if block != nil {
			if block.Type != "CERTIFICATE" {
				return fmt.Errorf("failure loading %s: block %d does not contain a certificate", configPath, pemBlockNum)
			}
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return fmt.Errorf("failure loading %s: failed to parse CA certificate in block %d", configPath, pemBlockNum)
			}
			f.caList = append(f.caList, cert)
			pemBlockNum++
		} else {
			break
		}
	}
	return nil
}
