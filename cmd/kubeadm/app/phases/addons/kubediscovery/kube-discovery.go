/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-10-10  @author weiwei@tenxcloud.com
 */

package kubediscovery

import (
	"fmt"
	"time"
	"runtime"
	"crypto/x509"
	"crypto/rsa"
	"encoding/json"

	"k8s.io/api/core/v1"
	apps "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes/scheme"
	certutil "k8s.io/client-go/util/cert"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/apimachinery/pkg/util/wait"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/bootstrap/token/api"
	certsphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/certs"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/certs/pkiutil"
	"strings"
)

func CreateKubeDiscoveryAddon(cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {
	//PHASE 1: parsing kube-discovery daemonset template create kube-discovery containers
	daemonSetBytes, err := kubeadmutil.ParseTemplate(KubeDiscovery, struct{ ImageRepository, Arch, Version, SecretName string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         Version,
		SecretName:      api.ConfigMapClusterInfo,
	})
	if err != nil {
		return fmt.Errorf("error when parsing kube-discovery daemonset template: %v", err)
	}
	//PHASE 2: create cluster-info secret
	if err = createClusterInfoSecret(cfg, client); err != nil {
		return fmt.Errorf("error when creating cluster-info secret: %v", err)
	}
	//PHASE 3: create kube-discovery containers
	if err := createKubeDiscovery(daemonSetBytes, client); err != nil {
		return err
	}
	fmt.Println("[addons] Applied essential addon: kube-discovery")
	return nil
}

func createKubeDiscovery(daemonSetBytes []byte, client clientset.Interface) error {
	//PHASE 1: create kube-discovery daemonSet
	daemonSet := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), daemonSetBytes, daemonSet); err != nil {
		return fmt.Errorf("unable to decode kube-discovery daemonset %v", err)
	}
	// Create the DaemonSet for kube-discovery or update it in case it already exists
	if err := apiclient.CreateOrUpdateDaemonSet(client, daemonSet); err != nil {
		return fmt.Errorf("error when create kube-discovery daemonset: %v", err)
	}
	fmt.Println("[addons] Applied essential addon: kube-discovery, waiting for it to become ready")
	start := time.Now()
	wait.PollInfinite(constants.APICallRetryInterval, func() (bool, error) {
		d, err := client.AppsV1().DaemonSets(daemonSet.Namespace).Get(daemonSet.Name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		if d.Status.CurrentNumberScheduled != d.Status.DesiredNumberScheduled {
			return false, nil
		}
		return true, nil
	})
	fmt.Printf("[addons] Essential addon: kube-discovery is ready after %f seconds\n", time.Since(start).Seconds())
	return nil
}

func createClusterInfoSecret(cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {
	//PHASE 1: load ca.crt & ca.key Certificate Authority
	caCert, caKey, err := certsphase.LoadCertificateAuthority(cfg.CertificatesDir, constants.CACertAndKeyBaseName)
	if err != nil {
		return fmt.Errorf("unable to load CA Certificate for cluster-info secret from %s; %v", cfg.CertificatesDir, err)
	}
	etcdCaCert, etcdCaKey, err := certsphase.LoadCertificateAuthority(cfg.CertificatesDir, constants.EtcdCACertAndKeyBaseName)
	if err != nil {
		return fmt.Errorf("unable to load Etcd CA Certificate for cluster-info secret from %s; %v", cfg.CertificatesDir, err)
	}
	frontProxyCaCert, frontProxyCaKey, err := certsphase.LoadCertificateAuthority(cfg.CertificatesDir, constants.FrontProxyCACertAndKeyBaseName)
	if err != nil {
		return fmt.Errorf("unable to load Front Proxy CA Certificate for cluster-info secret from %s; %v", cfg.CertificatesDir, err)
	}
	//PHASE 2: try and load sa.key Service Account Key
	saKey, err := pkiutil.TryLoadKeyFromDisk(cfg.CertificatesDir, constants.ServiceAccountKeyBaseName)
	if err != nil {
		// If there's no sa.key Service Account Key, make sure every private key exists.
		return fmt.Errorf("unable to load service account key; %v", err)
	}
	//PHASE 3: encode cluster-info secret data
	secretData, err := encodeClusterInfoSecretData(cfg, caCert,etcdCaCert,frontProxyCaCert, caKey,etcdCaKey,frontProxyCaKey, saKey)
	if err != nil {
		return fmt.Errorf("unable to encode cluster-info secret; %v", err)
	}
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      api.ConfigMapClusterInfo,
			Namespace: metav1.NamespaceSystem,
		},
		Type: v1.SecretTypeOpaque,
		Data: secretData,
	}
	//PHASE 4: create cluster-info secret
	err = apiclient.CreateOrUpdateSecret(client, secret)
	if err != nil {
		return fmt.Errorf("unable to create cluster-info secret; %v", err)
	}
	return nil
}

func encodeClusterInfoSecretData(cfg *kubeadmapi.InitConfiguration, caCert,etcdCaCert,frontProxyCaCert *x509.Certificate, caKey,etcdCaKey,frontProxyCaKey *rsa.PrivateKey, saKey *rsa.PrivateKey) (map[string][]byte, error) {
	var (
		data      = map[string][]byte{}
		endpoints = []string{}           // ["https://192.168.1.235:6443","https://192.168.1.236:6443"]
		tokenMap  = map[string]string{}  // {"y5px6k":"mhssfuxqtolum2yt"}
		err       error
	)
	endpoints = append(endpoints, fmt.Sprintf("https://%s:%d", cfg.APIEndpoint.AdvertiseAddress, cfg.APIEndpoint.BindPort))
	for _, bt := range cfg.BootstrapTokens {
		tokenMap[bt.Token.ID] = bt.Token.Secret
	}
	data["endpoint-list.json"], err = json.Marshal(endpoints)
	if err != nil {
		return nil, fmt.Errorf("unable to encode endpoint-list.json; %v", err)
	}
	data["token-map.json"], err = json.Marshal(tokenMap)
	if err != nil {
		return nil, fmt.Errorf("unable to encode token-map.json; %v", err)
	}
	// ca.crt && ca.key
	data[constants.CACertName] = certutil.EncodeCertPEM(caCert)
	data[constants.CAKeyName] = certutil.EncodePrivateKeyPEM(caKey)
    // etcd/ca.crt && etcd/ca.key
	data[strings.Replace(constants.EtcdCACertName,"/","-",-1)] = certutil.EncodeCertPEM(etcdCaCert)
    data[strings.Replace(constants.EtcdCAKeyName,"/","-",-1)] =  certutil.EncodePrivateKeyPEM(etcdCaKey)
    // front-proxy-ca.crt && front-proxy-ca.key
    data[constants.FrontProxyCACertName] = certutil.EncodeCertPEM(frontProxyCaCert)
    data[constants.FrontProxyCAKeyName]  = certutil.EncodePrivateKeyPEM(frontProxyCaKey)
	// sa.key && sa.pub
	data[constants.ServiceAccountPrivateKeyName] = certutil.EncodePrivateKeyPEM(saKey)
	return data, nil
}
