/*
* Copyright 2014 Jason Woods.
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
*
* Derived from Golang src/pkg/crypto/tls/generate_cert.go
* Copyright 2009 The Go Authors. All rights reserved.
* Use of this source code is governed by a BSD-style
* license that can be found in the LICENSE file.
 */

package main

import (
  "bufio"
  "crypto/rand"
  "crypto/rsa"
  "crypto/x509"
  "crypto/x509/pkix"
  "encoding/pem"
  "fmt"
  "math/big"
  "net"
  "os"
  "strconv"
  "time"
)

var input *bufio.Reader

func init() {
  input = bufio.NewReader(os.Stdin)
}

func readString(prompt string) string {
  fmt.Printf("%s: ", prompt)

  var line []byte
  for {
    data, prefix, _ := input.ReadLine()
    line = append(line, data...)
    if !prefix {
      break
    }
  }

  return string(line)
}

func readNumber(prompt string) (num int64) {
  var err error
  for {
    if num, err = strconv.ParseInt(readString(prompt), 0, 64); err != nil {
      fmt.Println("Please enter a valid numerical value")
      continue
    }
    break
  }
  return
}

func anyKey() {
  input.ReadRune()
}

func main() {
  var err error

  template := x509.Certificate{
    Subject: pkix.Name{
      Organization: []string{"Log Courier"},
    },
    NotBefore: time.Now(),

    KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
    ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
    BasicConstraintsValid: true,

    IsCA: true,
  }

  fmt.Println("Specify the Common Name for the certificate. The common name")
  fmt.Println("can be anything, but is usually set to the server's primary")
  fmt.Println("DNS name. Even if you plan to connect via IP address you")
  fmt.Println("should specify the DNS name here.")
  fmt.Println()

  template.Subject.CommonName = readString("Common name")
  fmt.Println()

  fmt.Println("The next step is to add any additional DNS names and IP")
  fmt.Println("addresses that clients may use to connect to the server. If")
  fmt.Println("you plan to connect to the server via IP address and not DNS")
  fmt.Println("then you must specify those IP addresses here.")
  fmt.Println("When you are finished, just press enter.")
  fmt.Println()

  var cnt = 0
  var val string
  for {
    cnt++

    if val = readString(fmt.Sprintf("DNS or IP address %d", cnt)); val == "" {
      break
    }

    if ip := net.ParseIP(val); ip != nil {
      template.IPAddresses = append(template.IPAddresses, ip)
    } else {
      template.DNSNames = append(template.DNSNames, val)
    }
  }

  fmt.Println()

  fmt.Println("How long should the certificate be valid for? A year (365")
  fmt.Println("days) is usual but requires the certificate to be regenerated")
  fmt.Println("within a year or the certificate will cease working.")
  fmt.Println()

  template.NotAfter = template.NotBefore.Add(time.Duration(readNumber("Number of days")) * time.Hour * 24)

  fmt.Println("Common name:", template.Subject.CommonName)
  fmt.Println("DNS SANs:")
  if len(template.DNSNames) == 0 {
    fmt.Println("    None")
  } else {
    for _, e := range template.DNSNames {
      fmt.Println("   ", e)
    }
  }
  fmt.Println("IP SANs:")
  if len(template.IPAddresses) == 0 {
    fmt.Println("    None")
  } else {
    for _, e := range template.IPAddresses {
      fmt.Println("   ", e)
    }
  }
  fmt.Println()

  fmt.Println("The certificate can now be generated")
  fmt.Println("Press any key to begin generating the self-signed certificate.")
  anyKey()

  priv, err := rsa.GenerateKey(rand.Reader, 2048)
  if err != nil {
    fmt.Println("Failed to generate private key:", err)
    os.Exit(1)
  }

  serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
  template.SerialNumber, err = rand.Int(rand.Reader, serialNumberLimit)
  if err != nil {
    fmt.Println("Failed to generate serial number:", err)
    os.Exit(1)
  }

  derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
  if err != nil {
    fmt.Println("Failed to create certificate:", err)
    os.Exit(1)
  }

  certOut, err := os.Create("selfsigned.crt")
  if err != nil {
    fmt.Println("Failed to open selfsigned.pem for writing:", err)
    os.Exit(1)
  }
  pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
  certOut.Close()

  keyOut, err := os.OpenFile("selfsigned.key", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
  if err != nil {
    fmt.Println("failed to open selfsigned.key for writing:", err)
    os.Exit(1)
  }
  pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
  keyOut.Close()

  fmt.Println("Successfully generated certificate")
  fmt.Println("    Certificate: selfsigned.crt")
  fmt.Println("    Private Key: selfsigned.key")
  fmt.Println()
  fmt.Println("Copy and paste the following into your Log Courier")
  fmt.Println("configuration, adjusting paths as necessary:")
  fmt.Println("    \"transport\": \"tls\",")
  fmt.Println("    \"ssl ca\":    \"path/to/selfsigned.crt\",")
  fmt.Println()
  fmt.Println("Copy and paste the following into your LogStash configuration, ")
  fmt.Println("adjusting paths as necessary:")
  fmt.Println("    ssl_certificate => \"path/to/selfsigned.crt\",")
  fmt.Println("    ssl_key         => \"path/to/selfsigned.key\",")
}
