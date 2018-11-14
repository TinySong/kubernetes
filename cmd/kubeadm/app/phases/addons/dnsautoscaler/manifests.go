/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-10-10  @author weiwei@tenxcloud.com
 */
package dnsautoscaler


/*
 *
 * DNS Horizontal Autoscaler
 *
 * DNS Horizontal Autoscaler enables horizontal autoscaling feature for DNS service in Kubernetes clusters.
 * This autoscaler runs as a Deployment. It collects cluster status from the APIServer,
 * horizontally scales the number of DNS backends based on demand.
 * Autoscaling parameters could be tuned by modifying the kube-dns-autoscaler ConfigMap in kube-system namespace.
 *
 * gcr.io/google-containers/cluster-proportional-autoscaler-amd64:1.2.0
 *
 * http://kubernetes.io/docs/tasks/administer-cluster/dns-horizontal-autoscaling/
 * https://github.com/kubernetes-incubator/cluster-proportional-autoscaler/
 * https://github.com/kubernetes/kubernetes/blob/release-1.12/cluster/addons/dns-horizontal-autoscaler/dns-horizontal-autoscaler.yaml
 *
 */



const (


	KubeDnsAutoscalerVersion         = "1.2.0"

	KubeDnsAutoscaler = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-dns-autoscaler
  namespace: kube-system
  labels:
    k8s-app: kube-dns-autoscaler
    kubernetes.io/cluster-service: "true"
spec:
  selector:
    matchLabels:
      k8s-app: kube-dns-autoscaler
  template:
    metadata:
      labels:
        k8s-app: kube-dns-autoscaler
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ''
        seccomp.security.alpha.kubernetes.io/pod: 'docker/default'
    spec:
      priorityClassName: system-cluster-critical
      containers:
      - name: autoscaler
        image: {{ .ImageRepository }}/cluster-proportional-autoscaler-{{ .Arch }}:{{ .Version }}
        resources:
            requests:
                cpu: "20m"
                memory: "10Mi"
        command:
          - /cluster-proportional-autoscaler
          - --namespace=kube-system
          - --configmap=kube-dns-autoscaler
          - --target={{.Target}}
          - --default-params={"linear":{"coresPerReplica":256,"nodesPerReplica":16,"preventSinglePointFailure":true}}
          - --logtostderr=true
          - --v=2
      tolerations:
      - effect: NoSchedule
        operator: Exists
      - effect: NoExecute
        operator: Exists
      serviceAccountName: kube-dns-autoscaler
`

	// for kube-dns-autoscaler
	ServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-dns-autoscaler
  namespace: kube-system
`


	ClusterRole=`
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:kube-dns-autoscaler
  labels:
    addonmanager.kubernetes.io/mode: Reconcile
rules:
  - apiGroups: [""]
    resources: ["nodes"]
    verbs: ["get","list","watch"]
  - apiGroups: [""]
    resources: ["replicationcontrollers/scale"]
    verbs: ["get", "update"]
  - apiGroups: ["extensions"]
    resources: ["deployments/scale", "replicasets/scale"]
    verbs: ["get", "update"]
  - apiGroups: ["apps"]
    resources: ["deployments/scale", "replicasets/scale"]
    verbs: ["get", "update"]
  - apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["get", "create"]
`

	ClusterRoleBinding = `
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:kube-dns-autoscaler
subjects:
  - kind: ServiceAccount
    name: kube-dns-autoscaler
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: system:kube-dns-autoscaler
  apiGroup: rbac.authorization.k8s.io
`

)
