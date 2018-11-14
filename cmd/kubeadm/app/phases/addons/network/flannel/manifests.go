/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-10-10  @author weiwei@tenxcloud.com
 */
package flannel


/**
 *  quay.io/coreos/flannel:v0.10.0
 *  quay.io/coreos/flannel:v0.10.0-amd64
 *  https://github.com/coreos/flannel/blob/master/Documentation/configuration.md
 *  https://github.com/coreos/flannel/blob/master/Documentation/backends.md
 *  https://github.com/coreos/flannel/blob/master/Documentation/kube-flannel.yml
 */

const (

	Version = "v0.10.0"

	ConfigMap = `
kind: ConfigMap
apiVersion: v1
metadata:
  name: flannel
  namespace: kube-system
  labels:
    tier: node
    app: flannel
data:
  cni-conf.json: |
    {
      "name": "k8s",
      "cniVersion":"0.3.1",
      "plugins": [
        {
          "type": "flannel",
          "delegate": {
            "isDefaultGateway": true
          }
        },{
           "type": "tuning",
           "sysctl": {
               "net.core.somaxconn": "512"
            }
        },{
           "type": "bandwidth",
           "capabilities": {
              "bandwidth": true
            }
        }
      ]
    }
  net-conf.json: |
    {
      "Network": {{ .PodSubnet }},
      "Backend": {
        "Type": {{ .Backend }}
      }
    }`

	DaemonSet = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: flannel
  namespace: kube-system
  labels:
    tier: node
    app: flannel
spec:
  selector:
    matchLabels:
      tier: node
      app: flannel
  template:
    metadata:
      labels:
        tier: node
        app: flannel
    spec:
      hostNetwork: true
      nodeSelector:
        beta.kubernetes.io/arch: amd64
        beta.kubernetes.io/os: linux
      tolerations:
        - effect: NoSchedule
          operator: Exists
        - effect: NoExecute
          operator: Exists
      serviceAccountName: flannel
      initContainers:
      - name: install-cni
        image: {{ .ImageRepository }}/flannel:{{ .Version }}
        command:
        - cp
        args:
        - -f
        - /etc/kube-flannel/cni-conf.json
        - /etc/cni/net.d/10-flannel.conflist
        volumeMounts:
        - name: cni
          mountPath: /etc/cni/net.d
        - name: config
          mountPath: /etc/kube-flannel/
      containers:
      - name: flannel
        image: {{ .ImageRepository }}/flannel:{{ .Version }}
        command:
        - /opt/bin/flanneld
        args:
        - --etcd-endpoints=https://kubernetes.cluster.local:2379
        - --etcd-cafile=/etc/kubernetes/pki/etcd/ca.crt
        - --etcd-certfile=/etc/kubernetes/pki/etcd/client.crt
        - --etcd-keyfile=/etc/kubernetes/pki/etcd/client.key
        - --ip-masq
        resources:
          requests:
            cpu: "100m"
            memory: "50Mi"
          limits:
            cpu: "100m"
            memory: "50Mi"
        securityContext:
          privileged: true
        env:
        - name: POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        volumeMounts:
        - name: run
          mountPath: /run
        - name: config
          mountPath: /etc/kube-flannel/
        - name: pki
          mountPath: /etc/kubernetes/pki
          readOnly: true
        - name: etc-resolv-conf
          mountPath: /etc/resolv.conf
          readOnly: true
      volumes:
        - name: run
          hostPath:
            path: /run
            type: DirectoryOrCreate
        - name: cni
          hostPath:
            path: /etc/cni/net.d
            type: DirectoryOrCreate
        - name: pki
          hostPath:
            path: /etc/kubernetes/pki
            type: DirectoryOrCreate
        - name: etc-resolv-conf
          hostPath:
            path: /etc/resolv.conf
            type: FileOrCreate
        - name: config
          configMap:
            name: flannel
`

	//This manifest deploys a Job which performs one time configuration of flannel.
	Job = `
apiVersion: batch/v1
kind: Job
metadata:
  name: configure-flannel
  namespace: kube-system
  labels:
    k8s-app: flannel
spec:
  template:
    metadata:
      name: configure-flannel
    spec:
      hostNetwork: true
      restartPolicy: OnFailure
      nodeSelector:
        node-role.kubernetes.io/master: ""
      tolerations:
        - effect: NoSchedule
          operator: Exists
        - effect: NoExecute
          operator: Exists
      containers:
        - name: configure-flannel
          image: {{ .Image }}
          command:
          - "etcdctl"
          - "--endpoints=http://127.0.0.1:2379"
          - "--no-sync"
          - "set"
          - "/coreos.com/network/config"
          - '{ "Network": "{{ .PodSubnet }}", "Backend": {"Type": "vxlan"} }'
`


	// for flannel
	ServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: flannel
  namespace: kube-system
`


	ClusterRole=`
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:flannel
rules:
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - get
  - apiGroups:
      - ""
    resources:
      - nodes
    verbs:
      - list
      - watch
  - apiGroups:
      - ""
    resources:
      - nodes/status
    verbs:
      - patch
`

	ClusterRoleBinding = `
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:flannel
subjects:
  - kind: ServiceAccount
    name: flannel
    namespace: kube-system
roleRef:
  kind: ClusterRole
  name: system:flannel
  apiGroup: rbac.authorization.k8s.io
`

)
