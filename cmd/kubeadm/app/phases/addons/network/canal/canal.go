/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-02-05  @author weiwei@tenxcloud.com
 */
package canal


import (
	"fmt"

	"k8s.io/api/core/v1"
	batch "k8s.io/api/batch/v1"
	rbac "k8s.io/api/rbac/v1"
	apps "k8s.io/api/apps/v1"
	clientset "k8s.io/client-go/kubernetes"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/kubernetes/cmd/kubeadm/app/images"
)

func CreateCanalAddon(cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {
	//PHASE 1: create the per-node agents containers
	configMapBytes, err := kubeadmutil.ParseTemplate(ConfigMap, nil)
	if err != nil {
		return fmt.Errorf("error when parsing canal configmap template: %v", err)
	}
	daemonSetBytes, err := kubeadmutil.ParseTemplate(DaemonSet, struct{ ImageRepository, FlannelVersion, CalicoVersion string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		FlannelVersion: FlannelVersion,
		CalicoVersion : CalicoVersion,
	})
	if err != nil {
		return fmt.Errorf("error when parsing the per-node agents daemonset template: %v", err)
	}
	if err := createCanalNode(daemonSetBytes, configMapBytes, client); err != nil {
		return err
	}
	//PHASE 2: create kube controllers containers
	policyControllerDeploymentBytes, err := kubeadmutil.ParseTemplate(KubeController, struct{ ImageRepository, CalicoVersion string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		CalicoVersion : CalicoVersion,
	})
	if err != nil {
		return fmt.Errorf("error when parsing kube controllers deployment template: %v", err)
	}
	if err := createKubeControllers(policyControllerDeploymentBytes, client); err != nil {
		return err
	}
	//PHASE 3: create  etcdctl job to configure ip pool
	ctlJobBytes, err := kubeadmutil.ParseTemplate(Job, struct{ Image, PodSubnet string }{
		Image:      images.GetEtcdImage(&cfg.ClusterConfiguration),
		PodSubnet:  cfg.Networking.PodSubnet,
	})
	if err != nil {
		return fmt.Errorf("error when parsing etcdctl job template: %v", err)
	}
	if err := createEtcdCtl(ctlJobBytes, client); err != nil {
		return err
	}
	fmt.Println("[addons] Applied essential addon: canal")
	return nil
}

func createCanalNode(daemonSetBytes, configBytes []byte, client clientset.Interface) error {

	//PHASE 1: create ConfigMap for Canal CNI
	configMap := &v1.ConfigMap{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), configBytes, configMap); err != nil {
		return fmt.Errorf("unable to decode Canal CNI configmap %v", err)
	}

	// Create the ConfigMap for Canal CNI or update it in case it already exists
	if err := apiclient.CreateOrUpdateConfigMap(client, configMap); err != nil {
		return err
	}

	//PHASE 2: create RBAC rules
	clusterRoles := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRole), clusterRoles); err != nil {
		return fmt.Errorf("unable to decode canal clusterroles %v", err)
	}
	calicoClusterRole := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(CalicoClusterRole), calicoClusterRole); err != nil {
		return fmt.Errorf("unable to decode calico clusterroles %v", err)
	}
	flannelClusterRole := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(FlannelClusterRole), flannelClusterRole); err != nil {
		return fmt.Errorf("unable to decode flannel clusterroles %v", err)
	}

	// Create the ClusterRoles for Canal Node or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRoles); err != nil {
		return err
	}
	if err := apiclient.CreateOrUpdateClusterRole(client, calicoClusterRole); err != nil {
		return err
	}
	if err := apiclient.CreateOrUpdateClusterRole(client, flannelClusterRole); err != nil {
		return err
	}

	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode canal clusterrolebindings %v", err)
	}
	calicoClusterRoleBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(CalicoClusterRoleBinding), calicoClusterRoleBinding); err != nil {
		return fmt.Errorf("unable to decode calico clusterrolebindings %v", err)
	}
	flannelClusterRoleBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(FlannelClusterRoleBinding), flannelClusterRoleBinding); err != nil {
		return fmt.Errorf("unable to decode flannel clusterrolebindings %v", err)
	}

	// Create the ClusterRoleBindings for Canal Node or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		return err
	}
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, calicoClusterRoleBinding); err != nil {
		return err
	}
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, flannelClusterRoleBinding); err != nil {
		return err
	}

	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode Canal serviceAccount %v", err)
	}

	// Create the ServiceAccount for Canal or update it in case it already exists
	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	//PHASE 3: create calico daemonSet
	daemonSet := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), daemonSetBytes, daemonSet); err != nil {
		return fmt.Errorf("unable to decode Canal node daemonset %v", err)
	}

	// Create the DaemonSet for calico node or update it in case it already exists
	return apiclient.CreateOrUpdateDaemonSet(client, daemonSet)
}

func createKubeControllers(deploymentBytes []byte, client clientset.Interface) error {

	//PHASE 1: create RBAC rules
	clusterRoles := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(KubeControllersClusterRole), clusterRoles); err != nil {
		return fmt.Errorf("unable to decode kube controllers clusterroles %v", err)
	}

	// Create the ClusterRoles for kube controllers or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRoles); err != nil {
		return err
	}

	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(KubeControllersClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode kube controllers clusterrolebindings %v", err)
	}

	// Create the ClusterRoleBindings for kube controllers or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		return err
	}

	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(KubeControllersServiceAccount), serviceAccount); err != nil {
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

func createEtcdCtl(JobBytes []byte, client clientset.Interface) error {
	//PHASE 1 : create Job to configure flannel ip pool
	job := &batch.Job{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), JobBytes, job); err != nil {
		return fmt.Errorf("unable to decode configure flannel Job %v", err)
	}
	return apiclient.CreateOrUpdateJob(client, job)
}
