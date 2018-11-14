/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-10-10  @author weiwei@tenxcloud.com
 */
package serviceproxy

import (
	"fmt"

	"k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	apps "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes/scheme"
	clientset "k8s.io/client-go/kubernetes"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
)

func CreateTenxProxyAddon(cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {

	tenxProxyConfigMapBytes, err := kubeadmutil.ParseTemplate(TenxProxyDomainConfigMap, nil)
	if err != nil {
		return fmt.Errorf("error when parsing service-proxy kube-config configmap template: %v", err)
	}

	tenxProxyCertsConfigMapBytes, err := kubeadmutil.ParseTemplate(TenxProxyCertsConfigMap, nil)
	if err != nil {
		return fmt.Errorf("error when parsing service-proxy kube-certs configmap template: %v", err)
	}

	tenxProxyDaemonSetBytes, err := kubeadmutil.ParseTemplate(TenxProxyDaemonSet, struct{ ImageRepository, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Version:         TenxProxyVersion,
	})
	if err != nil {
		return fmt.Errorf("error when parsing service-proxy daemonset template: %v", err)
	}
	err = createTenxProxy(tenxProxyCertsConfigMapBytes, tenxProxyConfigMapBytes, tenxProxyDaemonSetBytes, client)
	if err != nil {
		return err
	}
	fmt.Println("[addons] Applied essential addon: service-proxy")
	return nil
}

func createTenxProxy(certsConfigMapBytes, configMapBytes, daemonSetBytes []byte, client clientset.Interface) error {
	//PHASE 1: create ConfigMap for service proxy
	certsConfigMap := &v1.ConfigMap{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), certsConfigMapBytes, certsConfigMap); err != nil {
		return fmt.Errorf("unable to decode tenx-proxy kube-certs configmap %v", err)
	}
	// Create the ConfigMap for Calico CNI or update it in case it already exists
	if err := apiclient.CreateOrUpdateConfigMap(client, certsConfigMap); err != nil {
		return err
	}

	tenxProxyConfigMap := &v1.ConfigMap{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), configMapBytes, tenxProxyConfigMap); err != nil {
		return fmt.Errorf("unable to decode tenx-proxy kube-config configmap %v", err)
	}
	// Create the ConfigMap for Calico CNI or update it in case it already exists
	if err := apiclient.CreateOrUpdateConfigMap(client, tenxProxyConfigMap); err != nil {
		return err
	}
	//PHASE 2: create RBAC rules
	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode kubectl clusterrolebindings %v", err)
	}

	// Create the ClusterRoleBindings for kubectl or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		return err
	}

	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode kubectl serviceAccount %v", err)
	}

	// Create the ConfigMap for kubectl or update it in case it already exists
	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	//PHASE 3: create service proxy daemonSet
	daemonSet := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), daemonSetBytes, daemonSet); err != nil {
		return fmt.Errorf("unable to decode service proxy daemonset %v", err)
	}

	// Create the DaemonSet for calico node or update it in case it already exists
	return apiclient.CreateOrUpdateDaemonSet(client, daemonSet)
}