/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-10-10  @author weiwei@tenxcloud.com
 */
package calico

import (
	"fmt"

	"k8s.io/api/core/v1"
	batch "k8s.io/api/batch/v1"
	rbac "k8s.io/api/rbac/v1"
	apps "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes/scheme"
	clientset "k8s.io/client-go/kubernetes"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
)


func CreateCalicoAddon(cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {
	//PHASE 1: create calico node containers
	nodeConfigMapBytes, err := kubeadmutil.ParseTemplate(NodeConfigMap, nil)
	if err != nil {
		return fmt.Errorf("error when parsing calico cni configmap template: %v", err)
	}
	cniDaemonSetBytes, err := kubeadmutil.ParseTemplate(Node, struct{ ImageRepository, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Version: Version,
	})
	if err != nil {
		return fmt.Errorf("error when parsing calico cni daemonset template: %v", err)
	}
	if err := createCalicoNode(cniDaemonSetBytes, nodeConfigMapBytes, client); err != nil {
		return err
	}
	//PHASE 2: create calico kube controllers containers
	policyControllerDeploymentBytes, err := kubeadmutil.ParseTemplate(KubeController, struct{ ImageRepository, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Version: Version,
	})
	if err != nil {
		return fmt.Errorf("error when parsing kube controllers deployment template: %v", err)
	}
	if err := createKubeControllers(policyControllerDeploymentBytes, client); err != nil {
		return err
	}
	//PHASE 3: create calico ctl job to configure ip pool
	ctlConfigMapBytes, err := kubeadmutil.ParseTemplate(CtlConfigMap, struct{ ServiceSubnet,PodSubnet string }{
		ServiceSubnet: cfg.Networking.ServiceSubnet,
		PodSubnet: cfg.Networking.PodSubnet,
	})
	if err != nil {
		return fmt.Errorf("error when parsing calicoctl configmap template: %v", err)
	}

	ctlJobBytes, err := kubeadmutil.ParseTemplate(CtlJob, struct{ ImageRepository, Version string }{
		ImageRepository:     cfg.GetControlPlaneImageRepository(),
		Version: Version,
	})
	if err != nil {
		return fmt.Errorf("error when parsing calicoctl job template: %v", err)
	}
	if err := createCalicoCtl(ctlJobBytes, ctlConfigMapBytes, client); err != nil {
		return err
	}
	fmt.Println("[addons] Applied essential addon: calico")
	return nil
}

func createCalicoNode(daemonSetBytes, configBytes []byte, client clientset.Interface) error {

	//PHASE 1: create ConfigMap for calico CNI
	cniConfigMap := &v1.ConfigMap{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), configBytes, cniConfigMap); err != nil {
		return fmt.Errorf("unable to decode Calico CNI configmap %v", err)
	}

	// Create the ConfigMap for Calico CNI or update it in case it already exists
	if err := apiclient.CreateOrUpdateConfigMap(client, cniConfigMap); err != nil {
		return err
	}

	//PHASE 2: create RBAC rules
	clusterRoles := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(CalicoClusterRole), clusterRoles); err != nil {
		return fmt.Errorf("unable to decode calico node clusterroles %v", err)
	}

	// Create the ClusterRoles for Calico Node or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRoles); err != nil {
		return err
	}

	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(CalicoClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode calico node clusterrolebindings %v", err)
	}

	// Create the ClusterRoleBindings for Calico Node or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		return err
	}

	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(CalicoServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode calico node serviceAccount %v", err)
	}

	// Create the ConfigMap for CoreDNS or update it in case it already exists
	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	//PHASE 3: create calico daemonSet
	daemonSet := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), daemonSetBytes, daemonSet); err != nil {
		return fmt.Errorf("unable to decode calico node daemonset %v", err)
	}

	// Create the DaemonSet for calico node or update it in case it already exists
	return apiclient.CreateOrUpdateDaemonSet(client, daemonSet)
}

func createKubeControllers(deploymentBytes []byte, client clientset.Interface) error {

	//PHASE 1: create RBAC rules
	clusterRoles := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(CalicoControllersClusterRole), clusterRoles); err != nil {
		return fmt.Errorf("unable to decode kube controllers clusterroles %v", err)
	}

	// Create the ClusterRoles for kube controllers or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRoles); err != nil {
		return err
	}

	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(CalicoControllersClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode kube controllers clusterrolebindings %v", err)
	}

	// Create the ClusterRoleBindings for kube controllers or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		return err
	}

	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(CalicoControllersServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode kube controllers serviceAccount %v", err)
	}

	// Create the ServiceAccount for kube controller or update it in case it already exists
	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	//PHASE 2: create kube controller deployment
	deployment := &apps.Deployment{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), deploymentBytes, deployment); err != nil {
		return fmt.Errorf("unable to decode kube controllers daemonset %v", err)
	}

	// Create the DaemonSet for calico kube controllers or update it in case it already exists
	return apiclient.CreateOrUpdateDeployment(client, deployment)
}

func createCalicoCtl(JobBytes, configMapBytes []byte, client clientset.Interface) error {
	//PHASE 1: create ConfigMap for calico ctl
	configMap := &v1.ConfigMap{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), configMapBytes, configMap); err != nil {
		return fmt.Errorf("unable to decode calico ctl configmap %v", err)
	}

	// Create the ConfigMap for Calico CNI or update it in case it already exists
	if err := apiclient.CreateOrUpdateConfigMap(client, configMap); err != nil {
		return err
	}

	//PHASE 2 : create Job to configure calico ip pool
	job := &batch.Job{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), JobBytes, job); err != nil {
		return fmt.Errorf("unable to decode calicoctl Job %v", err)
	}
	return apiclient.CreateOrUpdateJob(client, job)
}
