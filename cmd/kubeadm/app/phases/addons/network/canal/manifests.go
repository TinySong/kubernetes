/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-10-10  @author weiwei@tenxcloud.com
 */


/**
 *  Calico Version v3.3.0
 *  https://docs.projectcalico.org/v3.2/getting-started/kubernetes/installation/flannel
 *  Selecting your datastore type and number of nodes
 *  1> etcd datastore
 *  2> Kubernetes API datastore—50 nodes or less   (without Typha)<beta>
 *  3> Kubernetes API datastore—more than 50 nodes (with Typha)<beta>
 *  https://docs.projectcalico.org/v3.2/releases/
 *  This manifest includes the following component versions:
 *    quay.io/calico/node:v3.3.0
 *    quay.io/calico/cni:v3.3.0               # https://github.com/projectcalico/cni-plugin/blob/master/k8s-install/scripts/install-cni.sh
 *    quay.io/calico/ctl:v3.3.0
 *    quay.io/calico/kube-controllers:v3.3.0  # using kube-controllers only if you're using the etcd Datastore
 *    quay.io/calico/typha:v3.3.0             # using Typha only if you're using the Kubernetes API Datastore and you have more than 50 Kubernetes nodes.
 *    quay.io/calico/routereflector:v0.6.3    # Calico v3.3 will support running calico/node in a route reflection mode
 *                                            # https://github.com/projectcalico/calico/issues/1745
 *
 *    quay.io/coreos/flannel:v0.10.0
 *    quay.io/coreos/flannel:v0.10.0-amd64
 *
 *    https://docs.projectcalico.org/v3.3/getting-started/kubernetes/installation/hosted/canal/rbac.yaml
 *    https://docs.projectcalico.org/v3.3/getting-started/kubernetes/installation/hosted/canal/canal.yaml
 */

package canal

const (

	FlannelVersion = "v0.10.0"
	CalicoVersion  = "v3.3.0"

	//This ConfigMap is used to configure a self-hosted Canal Installation.
	ConfigMap =`
kind: ConfigMap
apiVersion: v1
metadata:
  name: canal-config
  namespace: kube-system
data:
  etcd_endpoints: "https://kubernetes.cluster.local:2379"
  etcd_ca: "/etc/kubernetes/pki/etcd/ca.crt"
  etcd_cert: "/etc/kubernetes/pki/etcd/client.crt"
  etcd_key: "/etc/kubernetes/pki/etcd/client.key"
  canal_iface: ""
  masquerade: "true"
  cni_network_config: |-
    {
        "name": "canal",
        "cniVersion": "0.3.1",
        "plugins": [
            {
                "type": "flannel",
                "delegate": {
                    "type": "calico",
                    "include_default_routes": true,
                    "etcd_endpoints": "__ETCD_ENDPOINTS__",
                    "etcd_key_file": "__ETCD_KEY_FILE__",
                    "etcd_cert_file": "__ETCD_CERT_FILE__",
                    "etcd_ca_cert_file": "__ETCD_CA_CERT_FILE__",
                    "log_level": "__LOG_LEVEL__",
                    "policy": {
                        "type": "k8s"
                    },
                    "kubernetes": {
                        "kubeconfig": "/etc/kubernetes/kubelet.conf"
                    }
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
    }`





    // This manifest installs the per-node agents, as well as the CNI plugins and network config on
    // each master and worker node in a Kubernetes cluster.
	DaemonSet =`
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: canal-node
  namespace: kube-system
  labels:
    k8s-app: canal-node
spec:
  selector:
    matchLabels:
      k8s-app: canal-node
  template:
    metadata:
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ''
      labels:
        k8s-app: canal-node
    spec:
      tolerations:
        - effect: NoSchedule
          operator: Exists
        - effect: NoExecute
          operator: Exists
      nodeSelector:
        beta.kubernetes.io/os: linux
        beta.kubernetes.io/arch: amd64
      hostNetwork: true
      serviceAccountName: canal
      terminationGracePeriodSeconds: 0
      initContainers:
        - name: install-cni
          image: {{ .ImageRepository }}/cni:{{ .Version }}
          imagePullPolicy: IfNotPresent
          command: ["/install-cni.sh"]
          resources:
            requests:
              cpu: 10m
              memory: 50Mi
            limits:
              cpu: 10m
              memory: 50Mi
          env:
            - name: ETCD_ENDPOINTS
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: etcd_endpoints
            - name: CNI_CONF_ETCD_CERT
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: etcd_cert
            - name: CNI_CONF_ETCD_KEY
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: etcd_key
            - name: CNI_CONF_ETCD_CA
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: etcd_ca
            - name: CNI_NETWORK_CONFIG
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: cni_network_config
            - name: CNI_CONF_NAME
              value: "10-calico.conflist"
            - name: CNI_MTU
              value: "1440"
            - name: SLEEP
              value: "false"
          volumeMounts:
            - mountPath: /host/opt/cni/bin
              name: cni-bin-dir
            - mountPath: /host/etc/cni/net.d
              name: cni-net-dir
      containers:
        - name: flannel
          image: {{ .ImageRepository }}/flannel:{{ .FlannelVersion }}
          command:
          - /opt/bin/flanneld
          args:
          - --etcd-endpoints=https://kubernetes.cluster.local:2379
          - --etcd-cafile=/etc/kubernetes/pki/etcd/ca.crt
          - --etcd-certfile=/etc/kubernetes/pki/etcd/client.crt
          - --etcd-keyfile=/etc/kubernetes/pki/etcd/client.key
          - --ip-masq
          env:
            - name: FLANNELD_ETCD_ENDPOINTS
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_endpoints
            - name: ETCD_CA_CERT_FILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_ca
            - name: ETCD_KEY_FILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_key
            - name: ETCD_CERT_FILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_cert
            - name: FLANNELD_ETCD_CAFILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_ca
            - name: FLANNELD_ETCD_KEYFILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_key
            - name: FLANNELD_ETCD_CERTFILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_cert
            - name: FLANNELD_IFACE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: canal_iface
            - name: FLANNELD_IP_MASQ
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: masquerade
            - name: FLANNELD_SUBNET_FILE
              value: "/run/flannel/subnet.env"
            - name: POD_NAME
              valueFrom:
                fieldRef:
                  fieldPath: metadata.name
            - name: POD_NAMESPACE
              valueFrom:
                fieldRef:
                  fieldPath: metadata.namespace
          securityContext:
            privileged: true
          volumeMounts:
            - mountPath: /etc/resolv.conf
              name: etc-resolv-conf
              readOnly: true
            - mountPath: /run/flannel
              name: run-flannel
            - mountPath: /etc/kubernetes/pki
              name: k8s-certs
              readOnly: true
        - name: calico-node
          image: {{ .ImageRepository }}/node:{{ .CalicoVersion }}
          env:
            - name: ETCD_ENDPOINTS
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_endpoints
            - name: ETCD_CA_CERT_FILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_ca
            - name: ETCD_KEY_FILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_key
            - name: ETCD_CERT_FILE
              valueFrom:
                configMapKeyRef:
                  name: canal-config
                  key: etcd_cert
            - name: CALICO_NETWORKING_BACKEND
              value: "none"
            - name: CLUSTER_TYPE
              value: "k8s,canal"
            - name: CALICO_DISABLE_FILE_LOGGING
              value: "true"
            - name: CALICO_K8S_NODE_REF
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: NO_DEFAULT_POOLS
              value: "true"
            - name: FELIX_IPV6SUPPORT
              value: "false"
            - name: FELIX_HEALTHENABLED
              value: "true"
          securityContext:
            privileged: true
          resources:
            requests:
              cpu: 250m
          readinessProbe:
            exec:
              command:
              - /bin/calico-node
              - -felix-ready
            periodSeconds: 10
          volumeMounts:
            - mountPath: /lib/modules
              name: lib-modules
              readOnly: true
            - mountPath: /var/run/calico
              name: var-run-calico
              readOnly: false
            - mountPath: /var/lib/calico
              name: var-lib-calico
              readOnly: false
            - mountPath: /etc/kubernetes/pki
              name: k8s-certs
              readOnly: true
            - mountPath: /etc/resolv.conf
              name: etc-resolv-conf
              readOnly: true
      volumes:
        - name: lib-modules
          hostPath:
            path: /lib/modules
            type: DirectoryOrCreate
        - name: var-run-calico
          hostPath:
            path: /var/run/calico
            type: DirectoryOrCreate
        - name: var-lib-calico
          hostPath:
            path: /var/lib/calico
            type: DirectoryOrCreate
        - name: cni-bin-dir
          hostPath:
            path: /opt/cni/bin
            type: DirectoryOrCreate
        - name: cni-net-dir
          hostPath:
            path: /etc/cni/net.d
            type: DirectoryOrCreate
        - name: run-flannel
          hostPath:
            path: /run/flannel
            type: DirectoryOrCreate
        - name: etc-resolv-conf
          hostPath:
            path: /etc/resolv.conf
            type: FileOrCreate
        - name: k8s-certs
          hostPath:
            path: /etc/kubernetes/pki
            type: DirectoryOrCreate`

	// This manifest installs the calico/kube-controllers container on each master.
	// See https://github.com/projectcalico/kube-controllers
	//     https://github.com/kubernetes/contrib/tree/master/election
	KubeController = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-controller
  namespace: kube-system
  labels:
    k8s-app: kube-controller
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      k8s-app: kube-controller
  template:
    metadata:
      labels:
        k8s-app: kube-controller
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ''
    spec:
      hostNetwork: true
      nodeSelector:
        beta.kubernetes.io/os: linux
        beta.kubernetes.io/arch: amd64
      tolerations:
        - effect: NoSchedule
          operator: Exists
        - effect: NoExecute
          operator: Exists
      serviceAccountName: kube-controllers
      containers:
      - name: kube-controller
        image: {{ .ImageRepository }}/kube-controllers:{{ .CalicoVersion }}
        imagePullPolicy: IfNotPresent
        env:
          - name: ETCD_ENDPOINTS
            valueFrom:
              configMapKeyRef:
                name: canal-config
                key: etcd_endpoints
          - name: ETCD_CA_CERT_FILE
            valueFrom:
              configMapKeyRef:
                name: canal-config
                key: etcd_ca
          - name: ETCD_KEY_FILE
            valueFrom:
              configMapKeyRef:
                name: canal-config
                key: etcd_key
          - name: ETCD_CERT_FILE
            valueFrom:
              configMapKeyRef:
                name: canal-config
                key: etcd_cert
          - name: KUBECONFIG
            value: /etc/kubernetes/kubelet.conf
          - name: ENABLED_CONTROLLERS
            value: policy,namespace,workloadendpoint,node,serviceaccount
          - name: LOG_LEVEL
            value: warning
        readinessProbe:
          exec:
            command:
            - /usr/bin/check-status
            - -r
        volumeMounts:
          - mountPath: /etc/resolv.conf
            name: etc-resolv-conf
            readOnly: true
          - mountPath: /etc/kubernetes
            name: k8s-certs
            readOnly: true
      volumes:
        - name: etc-resolv-conf
          hostPath:
            path: /etc/resolv.conf
            type: FileOrCreate
        - name: k8s-certs
          hostPath:
            path: /etc/kubernetes
            type: DirectoryOrCreate
`


    //This manifest deploys a Job which performs one time configuration of Canal.
    Job = `
apiVersion: batch/v1
kind: Job
metadata:
  name: configure-canal
  namespace: kube-system
  labels:
    k8s-app: canal
spec:
  template:
    metadata:
      name: configure-canal
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
        - name: configure-canal
          image: {{ .Image }}
          command:
          - "etcdctl"
          - "--endpoints=http://127.0.0.1:2379"
          - "--no-sync"
          - "set"
          - "/coreos.com/network/config"
          - '{ "Network": "{{ .PodSubnet }}", "Backend": {"Type": "vxlan"} }'
`



	ClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:canal
rules:
  - apiGroups: [""]
    resources:
      - pods
      - nodes
      - namespaces
    verbs:
      - get
`

	CalicoClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: system:calico
rules:
  - apiGroups: [""]
    resources:
      - namespaces
    verbs:
      - get
      - list
      - watch
  - apiGroups: [""]
    resources:
      - pods/status
    verbs:
      - update
  - apiGroups: [""]
    resources:
      - pods
    verbs:
      - get
      - list
      - watch
  - apiGroups: [""]
    resources:
      - nodes
    verbs:
      - get
      - list
      - update
      - watch
  - apiGroups: ["extensions"]
    resources:
      - networkpolicies
    verbs:
      - get
      - list
      - watch
  - apiGroups: ["crd.projectcalico.org"]
    resources:
      - globalfelixconfigs
      - bgppeers
      - globalbgpconfigs
      - ippools
      - globalnetworkpolicies
    verbs:
      - create
      - get
      - list
      - update
      - watch
`

	FlannelClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1beta1
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

	ServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: canal
  namespace: kube-system
`

	ClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:canal
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:canal
subjects:
- kind: ServiceAccount
  name: canal
  namespace: kube-system
`
	FlannelClusterRoleBinding = `
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: system:canal-flannel
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:flannel
subjects:
- kind: ServiceAccount
  name: canal
  namespace: kube-system
`

	CalicoClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: ClusterRoleBinding
metadata:
  name: system:canal-calico
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:calico
subjects:
- kind: ServiceAccount
  name: canal
  namespace: kube-system
`


	// for calico/kube-controllers
	KubeControllersClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:kube-controllers
rules:
  - apiGroups: [""]
    resources:
      - pods
      - namespaces
      - nodes
    verbs:
      - watch
      - list
  - apiGroups:
      - networking.k8s.io
    resources:
      - networkpolicies
    verbs:
      - watch
      - list
`

	KubeControllersServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-controllers
  namespace: kube-system
`

	KubeControllersClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:kube-controllers
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:kube-controllers
subjects:
- kind: ServiceAccount
  name: kube-controllers
  namespace: kube-system
`

)