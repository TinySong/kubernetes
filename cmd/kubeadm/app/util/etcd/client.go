/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package etcd

import (
	"time"
	"io/ioutil"
	"crypto/tls"
	"crypto/x509"
	"github.com/coreos/etcd/clientv3"
)

// NewEtcdClient returns an *clientv3.Client with a connection to named machines.
func NewEtcdClient(endpoints []string, cert, key, caCert string) (*clientv3.Client, error) {
	var c *clientv3.Client
	var err error
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
	}
	if caCert != "" {
		certBytes, err := ioutil.ReadFile(caCert)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		if caCertPool.AppendCertsFromPEM(certBytes) {
			tlsConfig.RootCAs = caCertPool
		}
	}
	if cert != "" && key != "" {
		tlsCert, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{tlsCert}
	}

	cfg := clientv3.Config{
		Endpoints:               endpoints,
		DialTimeout:             3 * time.Second,
	}
	cfg.TLS = tlsConfig
	c, err = clientv3.New(cfg)
	if err != nil {
		return nil, err
	}
	return c, nil
}