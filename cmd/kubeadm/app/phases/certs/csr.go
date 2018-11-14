/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-10-10  @author weiwei@tenxcloud.com
 */
package certs

import (
	"io"
	"fmt"
	"sync"
	"time"
	"bytes"
	"strings"
	"net/http"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"gopkg.in/square/go-jose.v2"
	"k8s.io/apimachinery/pkg/util/wait"
	certutil "k8s.io/client-go/util/cert"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	pkiutil "k8s.io/kubernetes/cmd/kubeadm/app/phases/certs/pkiutil"
	kubeadmapiv1alpha3 "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm/v1alpha3"

)

const (
	DefaultDiscoveryServicePort int32 = 9898
)


func PerformTLSBootstrap(cfg *kubeadmapi.JoinConfiguration) error {
	_, err := runForEndpointsAndReturnFirst(cfg.DiscoveryTokenAPIServers,cfg.DiscoveryTimeout.Duration,func(endpoint string) (*rsa.PrivateKey, error) {
		accessToken := strings.Split(cfg.Token, ".")
		if len(accessToken) < 2 {
			return nil, fmt.Errorf("please provide valid token[%v]", accessToken)
		}
		requestURL := fmt.Sprintf("http://%s/cluster-info/v1?token-id=%s", endpoint, accessToken[0])
		req, err := http.NewRequest("GET", requestURL, nil)
		if err != nil {
			return nil, fmt.Errorf("[discovery] failed to consturct an HTTP request [%v]", err)
		}
		//fmt.Printf("[certificates] created cluster info discovery client, requesting info from %q\n", requestURL)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("[discovery] failed to request cluster info [%v]", err)
		}
		buf := new(bytes.Buffer)
		io.Copy(buf, res.Body)
		res.Body.Close()

		object, err := jose.ParseSigned(buf.String())
		if err != nil {
			return nil, fmt.Errorf("[discovery] failed to parse response as JWS object [%v]", err)
		}
		fmt.Println("[discovery] cluster info object received, verifying signature and contents using token")
		output, err := object.Verify([]byte(accessToken[1]))
		if err != nil {
			return nil, fmt.Errorf("[discovery] failed to verify JWS signature of received cluster info object [%v]", err)
		}
		fmt.Println("[discovery] cluster info object received, cluster info signature and contents are valid ")
		clusterInfo := ClusterInfo{}
		if err := json.Unmarshal(output, &clusterInfo); err != nil {
			return nil, fmt.Errorf("[discovery] failed to decode received cluster info object [%v]", err)
		}
		// ca.crt & ca.key
		if clusterInfo.APICertificateAuthority == "" {
			return nil, fmt.Errorf("[discovery] cluster info object is invalid - no root CA certificate and key found")
		}
		caCert, err := certutil.ParseCertsPEM([]byte(clusterInfo.APICertificateAuthority))
		if err != nil {
			return nil, fmt.Errorf("[discovery] failed to Parse ca cert [%v]", err)
		}
		if pkiutil.WriteCert(kubeadmapiv1alpha3.DefaultCertificatesDir, kubeadmconstants.CACertAndKeyBaseName, caCert[0]); err != nil {
			return nil, fmt.Errorf("[discovery] failed to Parse ca key [%v]", err)
		}
		// etcd/ca.crt & etcd/ca.key
		if clusterInfo.EtcdCertificateAuthority == "" || clusterInfo.EtcdCertificateKey == "" {
			return nil, fmt.Errorf("[discovery] cluster info object is invalid - no root CA certificate and key found")
		}
		etcdCaCert, err := certutil.ParseCertsPEM([]byte(clusterInfo.EtcdCertificateAuthority))
		if err != nil {
			return nil, fmt.Errorf("[discovery] failed to Parse etcd ca cert [%v]", err)
		}
		etcdCaKey, err := certutil.ParsePrivateKeyPEM([]byte(clusterInfo.EtcdCertificateKey))
		if err != nil {
			return nil, fmt.Errorf("[discovery] failed to Parse etcd ca key [%v]", err)
		}
		if pkiutil.WriteCert(kubeadmapiv1alpha3.DefaultCertificatesDir, kubeadmconstants.EtcdCACertAndKeyBaseName, etcdCaCert[0]); err != nil {
			return nil, fmt.Errorf("[discovery] failed to Parse etcd ca  [%v]", err)
		}
		// sign etcd client certificate with etcd ca
		certCfg := &certutil.Config {
			CommonName: kubeadmconstants.EtcdClientCertCommonName,
			Organization: []string{kubeadmconstants.NodesGroup},
			Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}
		etcdClientCert, etcdClientKey, err := pkiutil.NewCertAndKey(etcdCaCert[0], etcdCaKey.(*rsa.PrivateKey), certCfg)
		if err != nil {
			return nil, fmt.Errorf("[discovery] failed to Create etcd client certificate & key   [%v]", err)
		}
		if err := pkiutil.WriteCertAndKey(kubeadmapiv1alpha3.DefaultCertificatesDir, kubeadmconstants.EtcdClientCertAndKeyBaseName, etcdClientCert, etcdClientKey); err != nil {
			return nil, fmt.Errorf("[discovery] failure while saving %s certificate and key: %v", kubeadmconstants.EtcdClientCertAndKeyBaseName, err)
		}

		return nil, nil
		})
	if err != nil {
		return fmt.Errorf("[discovery] Perform discovery certificate failed [%v]", err)
	}
    return nil
}

// runForEndpointsAndReturnFirst loops the endpoints slice and let's the endpoints race for connecting to the master
func runForEndpointsAndReturnFirst(endpoints []string, discoveryTimeout time.Duration, signCertificateFunc func(string) (*rsa.PrivateKey, error) )  (*rsa.PrivateKey, error) {
	stopChan := make(chan struct{})
	var privateKey *rsa.PrivateKey
	var once sync.Once
	var wg sync.WaitGroup
	for _, endpoint := range endpoints {
		wg.Add(1)
		ep := strings.Split(endpoint,":")
		socket := fmt.Sprintf("%s:%d",ep[0],DefaultDiscoveryServicePort)
		go func(apiEndpoint string) {
			defer wg.Done()
			wait.Until(func() {
				fmt.Printf("[discovery] Trying to connect to kube-discovery server %q\n", apiEndpoint)
				key, err := signCertificateFunc(apiEndpoint)
				if err != nil {
					fmt.Printf("[discovery] Failed to connect to kube-discovery server %q: %v\n", apiEndpoint, err)
					return
				}
				fmt.Printf("[discovery] Successfully established connection with kube-discovery server %q\n", apiEndpoint)
				once.Do(func() {
					close(stopChan)
					privateKey = key
				})
			}, kubeadmconstants.DiscoveryRetryInterval, stopChan)
		}(socket)
	}

	select {
	case <-time.After(discoveryTimeout):
		close(stopChan)
		err := fmt.Errorf("abort connecting to kube-discovery server after timeout of %v", discoveryTimeout)
		fmt.Printf("[discovery] %v\n", err)
		wg.Wait()
		return nil, err
	case <-stopChan:
		wg.Wait()
		return privateKey, nil
	}
}