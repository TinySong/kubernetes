/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2019 TenxCloud. All Rights Reserved.
 * 2019-04-12  @author weiwei@tenxcloud.com
 */
package ovn

import (
	"fmt"
	apps "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
)

func CreateOvnAddon(cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {
	//PHASE 1: create  ovn configmap to configure ovn South/North Bound Database, northd, ovn-controller
	apiEndpoint := fmt.Sprintf("https://kubernetes.%s:%d", cfg.Networking.DNSDomain, cfg.APIEndpoint.BindPort)
	configMapBytes, err := kubeadmutil.ParseTemplate(ConfigMap, struct{ APIEndpoint, PodSubnet, ServiceSubnet string }{
		APIEndpoint:   apiEndpoint,
		PodSubnet:     cfg.Networking.PodSubnet,
		ServiceSubnet: cfg.Networking.ServiceSubnet,
	})
	if err != nil {
		return fmt.Errorf("error when parsing ovn configmap template: %v", err)
	}
	if err := createOvnConfig(configMapBytes, client); err != nil {
		return err
	}
	//PHASE 2: create ovn South/North Bound Database
	dbDeploymentBytes, err := kubeadmutil.ParseTemplate(DBDeploy, struct{ ImageRepository string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
	})
	if err != nil {
		return fmt.Errorf("error when parsing ovn South/North Bound Database template: %v", err)
	}
	if err := createOvnDB(dbDeploymentBytes, client); err != nil {
		return err
	}

	//PHASE 3: create ovn Northd
	northdDeploymentBytes, err := kubeadmutil.ParseTemplate(NorthdDeploy, struct{ ImageRepository string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
	})
	if err != nil {
		return fmt.Errorf("error when parsing ovn Northd template: %v", err)
	}
	if err := createOvnNorthd(northdDeploymentBytes, client); err != nil {
		return err
	}
	//PHASE 3: create ovn Openflow Controller
	daemonSetBytes, err := kubeadmutil.ParseTemplate(Controller, struct{ ImageRepository string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
	})
	if err != nil {
		return fmt.Errorf("error when parsing ovn Openflow Controller template: %v", err)
	}
	if err := createOvnController(daemonSetBytes, client); err != nil {
		return err
	}
	return nil
}

func createOvnConfig(configBytes []byte, client clientset.Interface) error {
	//PHASE 0: create namespace
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ovn-kubernetes",
		},
	}
	if _, err := client.CoreV1().Namespaces().Create(ns); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			fmt.Printf("unable to create namespace %s: %v\n", ns.Name, err)
		}
	}
	//PHASE 1: create RBAC rules
	clusterRoles := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRole), clusterRoles); err != nil {
		return fmt.Errorf("unable to decode ovn clusterroles %v", err)
	}
	// Create the ClusterRoles for ovn  or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRoles); err != nil {
		return err
	}

	clusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ClusterRoleBinding), clusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode ovn clusterrolebindings  %v", err)
	}
	// Create the ClusterRoleBindings for ovn or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, clusterRolesBinding); err != nil {
		return err
	}

	adminClusterRolesBinding := &rbac.ClusterRoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(AdminClusterRoleBinding), adminClusterRolesBinding); err != nil {
		return fmt.Errorf("unable to decode ovn adminclusterrolebindings %v", err)
	}
	// Create the ClusterRoleBindings for ovn or update it in case it already exists
	if err := apiclient.CreateOrUpdateClusterRoleBinding(client, adminClusterRolesBinding); err != nil {
		return err
	}

	serviceAccount := &v1.ServiceAccount{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(ServiceAccount), serviceAccount); err != nil {
		return fmt.Errorf("unable to decode flannel serviceAccount %v", err)
	}
	// Create the ServiceAccount for ovn or update it in case it already exists
	if err := apiclient.CreateOrUpdateServiceAccount(client, serviceAccount); err != nil {
		return err
	}

	//PHASE 2: create ConfigMap for ovn
	configMap := &v1.ConfigMap{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), configBytes, configMap); err != nil {
		return fmt.Errorf("unable to decode ovn configmap %v", err)
	}
	return apiclient.CreateOrUpdateConfigMap(client, configMap)
}

func createOvnDB(deploymentBytes []byte, client clientset.Interface) error {
	//PHASE 1: create ovn South/North Bound Database Service
	service := &v1.Service{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), []byte(DBService), service); err != nil {
		return fmt.Errorf("unable to decode ovn clusterroles %v", err)
	}
	// Create the Service for ovn  or update it in case it already exists
	if err := apiclient.CreateOrUpdateService(client, service); err != nil {
		return err
	}
	//PHASE 2: create ovn db deployment
	deployment := &apps.Deployment{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), deploymentBytes, deployment); err != nil {
		return fmt.Errorf("unable to decode ovn db deployment %v", err)
	}
	// Create the deployment for ovn db or update it in case it already exists
	return apiclient.CreateOrUpdateDeployment(client, deployment)
}

func createOvnNorthd(deploymentBytes []byte, client clientset.Interface) error {
	//PHASE 2: create ovn Northd deployment
	deployment := &apps.Deployment{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), deploymentBytes, deployment); err != nil {
		return fmt.Errorf("unable to decode ovn Northd deployment %v", err)
	}
	// Create the deployment for ovn Northd or update it in case it already exists
	return apiclient.CreateOrUpdateDeployment(client, deployment)
}

func createOvnController(daemonSetBytes []byte, client clientset.Interface) error {
	//PHASE 1: create Ovn Controller daemonSet
	daemonSet := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), daemonSetBytes, daemonSet); err != nil {
		return fmt.Errorf("unable to decode ovn controller daemonset %v", err)
	}
	// Create the DaemonSet for ovn controller or update it in case it already exists
	return apiclient.CreateOrUpdateDaemonSet(client, daemonSet)

}
