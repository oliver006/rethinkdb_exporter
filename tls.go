package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
)

func prepareTLSConfig(caFile, certFile, keyFile string) (*tls.Config, error) {
	config := new(tls.Config)

	if len(certFile) != 0 || len(keyFile) != 0 {
		if len(certFile) == 0 || len(keyFile) == 0 {
			return nil, errors.New("cert file and key file must be both specified")
		}

		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("TLS file load error: %v", err)
		}

		config.Certificates = []tls.Certificate{cert}
	}

	if len(caFile) != 0 {
		ca, err := ioutil.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("TLS CA file load error: %v", err)
		}

		cp := x509.NewCertPool()
		if !cp.AppendCertsFromPEM(ca) {
			return nil, fmt.Errorf("TLS credentials: failed to append ca")
		}

		config.RootCAs = cp
	}

	return config, nil
}
