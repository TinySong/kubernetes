/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-10-10  @author weiwei@tenxcloud.com
 */
package certs

type TypeMeta struct {
	Kind string `json:"kind,omitempty"`
	APIVersion string `json:"apiVersion,omitempty"`
}

// cluster info secret stored in k8s kube-system
type ClusterInfo struct {
	TypeMeta
	//ca.crt defines control plane CA certificate
	APICertificateAuthority          string `json:"apiCertificateAuthority,omitempty"`
	//ca.key defines control plane CA certificate private key
	APICertificateKey                string `json:"apiCertificateKey,omitempty"`

	//etcd/ca.crt defines etcd's CA certificate
	EtcdCertificateAuthority         string `json:"etcdCertificateAuthority,omitempty"`
	//etcd/ca.crt defines etcd's CA certificate private key
	EtcdCertificateKey               string `json:"etcdCertificateKey,omitempty"`

	//front-proxy-ca.crt  defines front proxy CA certificate
	FrontProxyCertificateAuthority   string `json:"frontProxyCertificateAuthority,omitempty"`
	//front-proxy-ca.crt  defines front proxy CA certificate private key
	FrontProxyCertificateKey         string `json:"frontProxyCertificateKey,omitempty"`

	// ServiceAccountPrivateKeyName defines SA private key base name
	ServiceAccountPrivateKey         string `json:"serviceAccountPrivateKey,omitempty"`

	// endpoints define control plane`s address
	Endpoints                        []string `json:"endpoints,omitempty"`
}
