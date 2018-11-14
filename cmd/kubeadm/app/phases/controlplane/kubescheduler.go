/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-10-10  @author weiwei@tenxcloud.com
 */
package controlplane

import (
	"fmt"
	"k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/pkg/api/legacyscheme"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"
)

/*
 (1) MostRequestedPriority
 (2) LeastRequestedPriority
 (3) LeastRequestedPriority BalancedResourceAllocation
 */

const (

	// policy.cfg structure
	// "k8s.io/kubernetes/pkg/scheduler/api"
	KubeScheduler = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: kube-scheduler
  namespace: kube-system
data:
  policy.cfg: |-
    {
        "kind" : "Policy",
        "apiVersion" : "v1",
        "predicates" : [
            {"name" : "PodFitsHostPorts"},
            {"name" : "PodFitsResources"},
            {"name" : "NoDiskConflict"},
            {"name" : "NoVolumeZoneConflict"},
            {"name" : "MatchNodeSelector"},
            {"name" : "MaxGCEPDVolumeCount"},
            {"name" : "MaxEBSVolumeCount"},
            {"name" : "MaxAzureDiskVolumeCount"},
            {"name" : "PodToleratesNodeTaints"},
            {"name" : "MatchInterPodAffinity"},
            {"name" : "HostName"}
            ],
        "priorities" : [
            {"name" : "LeastRequestedPriority", "weight" : 1},
            {"name" : "BalancedResourceAllocation", "weight" : 1},
            {"name" : "SelectorSpreadPriority", "weight" : 1},
            {"name" : "InterPodAffinityPriority", "weight" : 1},
            {"name" : "NodeAffinityPriority", "weight" : 1},
            {"name" : "NodePreferAvoidPodsPriority", "weight" : 1},
            {"name" : "TaintTolerationPriority", "weight" : 1},
            {"name" : "ImageLocalityPriority", "weight" : 1}
            ],
        "hardPodAffinitySymmetricWeight" : 10,
        "alwaysCheckAllPredicates" : false
     }`

)



// CreateConfigMapIfForScheduler creates the kube-scheduler ConfigMap
// in kube-system namespace if it doesn't exist already
func CreateConfigMapForKubeScheduler(client clientset.Interface) error {
	//PHASE 1: create kube-scheduler ConfigMap in kube-system namespace
	configMapBytes, err := kubeadmutil.ParseTemplate(KubeScheduler, nil)
	if err != nil {
		return fmt.Errorf("error when parsing kube-scheduler configmap template: %v", err)
	}
	configMap := &v1.ConfigMap{}
	if err := kuberuntime.DecodeInto(legacyscheme.Codecs.UniversalDecoder(), configMapBytes, configMap); err != nil {
		return fmt.Errorf("unable to decode kube-scheduler configmap %v", err)
	}
	// Create or update the ConfigMap in the kube-system namespace
	if err := apiclient.CreateOrUpdateConfigMap(client, configMap); err != nil {
		return fmt.Errorf("unable to create kube-scheduler configmap %v", err)
	}

	//Deprecated see k8s.io/plugin/pkg/auth/authorizer/rbac/bootstrappolicy/policy.go#436
	// rbacv1helpers.NewRule(Read...).Groups(legacyGroup).Resources("configmaps").RuleOrDie()

	//PHASE 2: modify kube-scheduler clusterrole to make kube-scheduler component can get kube-scheduler configmaps
	//var options metav1.GetOptions
	//if clusterRole, err := client.RbacV1().ClusterRoles().Get("system:kube-scheduler",options); err != nil {
	//	rule := rbachelper.NewRule("get").Groups("").Resources("configmaps").RuleOrDie()
	//	clusterRole.Rules = append(clusterRole.Rules,rule)
	//	if _, err := client.RbacV1().ClusterRoles().Update(clusterRole); err != nil {
	//		return fmt.Errorf("unable to update RBAC clusterrole: %v", err)
	//	}
	//	fmt.Println("[controlplane] Updated system:kube-scheduler clusterrole successfully.")
	//} else {
	//	return fmt.Errorf("unable to get kube-scheduler clusterrole %v", err)
	//}
	return nil
}