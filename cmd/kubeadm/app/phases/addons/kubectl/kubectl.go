/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-10-10  @author weiwei@tenxcloud.com
 */
package kubectl

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
)

func CreateKubectlAddon(cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {
	//PHASE 1: create kubectl containers
	daemonSetBytes, err := kubeadmutil.ParseTemplate(DaemonSet, struct{ ImageRepository, Arch, Version string }{
		ImageRepository: cfg.GetControlPlaneImageRepository(),
		Arch:            runtime.GOARCH,
		Version:         cfg.KubernetesVersion,
	})
	if err != nil {
		return fmt.Errorf("error when parsing kubectl daemonset template: %v", err)
	}
	if err := createKubeCtl(daemonSetBytes, client); err != nil {
		return err
	}
	fmt.Println("[addons] Applied essential addon: kubectl")
	return nil
}

func createKubeCtl(daemonSetBytes []byte, client clientset.Interface) error {
	//PHASE 1: create RBAC rules
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

	//PHASE 2: create kubectl daemonSet
	daemonSet := &apps.DaemonSet{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), daemonSetBytes, daemonSet); err != nil {
		return fmt.Errorf("unable to decode kubectl daemonset %v", err)
	}

	// Create the DaemonSet for kubectl or update it in case it already exists
	return apiclient.CreateOrUpdateDaemonSet(client, daemonSet)

}