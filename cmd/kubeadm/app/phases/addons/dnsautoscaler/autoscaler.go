/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-10-10  @author weiwei@tenxcloud.com
 */
package dnsautoscaler

import (
	"fmt"
	"runtime"

	"k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	apps "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes/scheme"
	clientset "k8s.io/client-go/kubernetes"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
	"k8s.io/kubernetes/cmd/kubeadm/app/features"
)

func CreateDnsAutoscalerAddOns(cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {
	var target string
	if features.Enabled(cfg.FeatureGates, features.CoreDNS) {
		target ="Deployment/coredns"
	} else {
		target ="Deployment/kube-dns"
	}
	deploymentBytes, err := kubeadmutil.ParseTemplate(KubeDnsAutoscaler, struct{ ImageRepository, Arch, Version,Target string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         KubeDnsAutoscalerVersion,
		Target:          target,
	})
	if err != nil {
		return fmt.Errorf("error when parsing kube dns autoscaler template: %v", err)
	}
	if err := createDnsAutoscaler(deploymentBytes, client); err != nil {
		return err
	}
	fmt.Println("[addons] Applied essential addon: dns-autoscaler")
	return nil
}

func createDnsAutoscaler(deploymentBytes []byte, client clientset.Interface) error {

	//PHASE 1: create RBAC rules
	clusterRoles := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRole), clusterRoles); err != nil {
		return fmt.Errorf("unable to decode kube dns autoscaler clusterroles %v", err)
	}

	// Create the ClusterRoles for kube dns autoscaler or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRoles); err != nil {
		return err
	}

	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode kube dns autoscaler clusterrolebindings %v", err)
	}

	// Create the ClusterRoleBindings for kube dns autoscaler or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		return err
	}

	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode kube dns autoscaler serviceAccount %v", err)
	}

	// Create the ConfigMap for kube dns autoscaler or update it in case it already exists
	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	deployment := &apps.Deployment{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), deploymentBytes, deployment); err != nil {
		return fmt.Errorf("unable to decode kube dns autoscaler deployment %v", err)
	}

	// Create the Deployment for kube-dns-autoscaler or update it in case it already exists
	return apiclient.CreateOrUpdateDeployment(client, deployment)
}
