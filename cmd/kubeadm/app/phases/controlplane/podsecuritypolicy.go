/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-10-10  @author weiwei@tenxcloud.com
 */
package controlplane

import (
	"fmt"
	policy "k8s.io/api/policy/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	"k8s.io/kubernetes/cmd/kubeadm/app/util/apiclient"

)

const (
	DefaultPodSecurityPolicy = `
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: default
  annotations:
    seccomp.security.alpha.kubernetes.io/allowedProfileNames: '*'
    users/annotations: "1. Allow all Capabilities; 2. Allow the use of commonly used volumes; 3. Prohibition of privilege pod"
spec:
  privileged: false
  allowPrivilegeEscalation: false
  allowedCapabilities:
  - '*'
  fsGroup:
    rule: RunAsAny
  hostIPC: false
  hostNetwork: false
  hostPID: false
  runAsUser:
    rule: RunAsAny
  seLinux:
    rule: RunAsAny
  supplementalGroups:
    rule: RunAsAny
  volumes:
  - secret
  - emptyDir
  - gitRepo
  - hostPath
  - configMap
  - downwardAPI
  - projected
  - persistentVolumeClaim
`

	PrivilegedPodSecurityPolicy = `
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: system
  annotations:
    seccomp.security.alpha.kubernetes.io/allowedProfileNames: '*'
    users/annotations: "system PodSecurityPolicy for Kubernetes cluster,Don't change it unless you know what you're doing."
spec:
  privileged: true
  allowPrivilegeEscalation: true
  allowedCapabilities: ['*']
  volumes: ['*']
  hostNetwork: true
  hostIPC: true
  hostPID: true
  hostPorts:
  - min: 0
    max: 65535
  runAsUser:
    rule: 'RunAsAny'
  seLinux:
    rule: 'RunAsAny'
  supplementalGroups:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
`
	PrivilegedKubeSystemClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:privileged
rules:
- apiGroups: ['policy']
  resources: ['podsecuritypolicies']
  verbs:     ['use']
  resourceNames: ['system']
`
	PrivilegedKubeSystemRoleBinding = `
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:privileged
  namespace: kube-system
roleRef:
  kind: ClusterRole
  name: system:privileged
  apiGroup: rbac.authorization.k8s.io
subjects:
- kind: Group
  name: system:masters
  apiGroup: rbac.authorization.k8s.io
- kind: Group
  name: system:nodes
  apiGroup: rbac.authorization.k8s.io
- kind: Group
  name: system:serviceaccounts:kube-system
  apiGroup: rbac.authorization.k8s.io
`
)

// CreateDefaultPodSecurityPolicy creates Default PodSecurityPolicy if it doesn't yet exist
func CreateDefaultPodSecurityPolicy(client clientset.Interface) error {
	//PHASE 1:  Create PodSecurityPolicy
	err := CreatePodSecurityPolicy(client, DefaultPodSecurityPolicy, PrivilegedPodSecurityPolicy)
	if err != nil {
		return err
	}
	//PHASE 2: create clusterRole
	clusterRoleBytes, err := kubeadmutil.ParseTemplate(PrivilegedKubeSystemClusterRole, nil)
	if err != nil {
		return fmt.Errorf("error when parsing privileged kube-system clusterRole template: %v", err)
	}
	clusterRole := &rbac.ClusterRole{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), clusterRoleBytes, clusterRole); err != nil {
		return fmt.Errorf("unable to decode privileged kube-system clusterRole %v", err)
	}
	// Create or update the clusterRole in the kube-system namespace
	if err := apiclient.CreateOrUpdateClusterRole(client, clusterRole); err != nil {
		return fmt.Errorf("unable to createprivileged kube-system clusterRole %v", err)
	}
	//PHASE 3: create Role Binding
	roleBindingBytes, err := kubeadmutil.ParseTemplate(PrivilegedKubeSystemRoleBinding, nil)
	if err != nil {
		return fmt.Errorf("error when parsing privileged kube-system roleBinding template: %v", err)
	}
	roleBinding := &rbac.RoleBinding{}
	if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), roleBindingBytes, roleBinding); err != nil {
		return fmt.Errorf("unable to decode privileged kube-system roleBinding %v", err)
	}
	// Create or update the roleBinding in the kube-system namespace
	if err := apiclient.CreateOrUpdateRoleBinding(client, roleBinding); err != nil {
		return fmt.Errorf("unable to create privileged kube-system roleBinding %v", err)
	}
	return nil
}

func CreatePodSecurityPolicy(client clientset.Interface, policies ...string) error {
	if len(policies) == 0 {
		return nil
	}
	for _, p := range policies {
		//PHASE 1:  Create PodSecurityPolicy
		pspBytes, err := kubeadmutil.ParseTemplate(p, nil)
		if err != nil {
			return fmt.Errorf("error when parsing default podSecurityPolicy template: %v", err)
		}
		psp := &policy.PodSecurityPolicy{}
		if err := kuberuntime.DecodeInto(scheme.Codecs.UniversalDecoder(), pspBytes, psp); err != nil {
			return fmt.Errorf("unable to decode default podSecurityPolicy %v", err)
		}
		if _, err := client.PolicyV1beta1().PodSecurityPolicies().Create(psp); err != nil {
			if !apierrors.IsAlreadyExists(err) {
				return fmt.Errorf("unable to create default podSecurityPolicy: %v", err)
			}
			if _, err := client.PolicyV1beta1().PodSecurityPolicies().Update(psp); err != nil {
				return fmt.Errorf("unable to update default podSecurityPolicy: %v", err)
			}
		}

	}
	return nil
}
