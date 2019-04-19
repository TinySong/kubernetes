/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-10-10  @author weiwei@tenxcloud.com
 */

/**
 *  Calico Version v3.4.0
 *  https://docs.projectcalico.org/v3.4/getting-started/kubernetes/installation/calico
 *  Selecting your datastore type and number of nodes
 *  1> etcd datastore
 *  2> Kubernetes API datastore—50 nodes or less   (without Typha)<beta>
 *  3> Kubernetes API datastore—more than 50 nodes (with Typha)<beta>
 *  https://docs.projectcalico.org/v3.4/releases/
 *  This manifest includes the following component versions:
 *    quay.io/calico/node:v3.4.0
 *    quay.io/calico/cni:v3.4.0               # https://github.com/projectcalico/cni-plugin/blob/master/k8s-install/scripts/install-cni.sh
 *    quay.io/calico/ctl:v3.4.0
 *    quay.io/calico/kube-controllers:v3.4.0  # using kube-controllers only if you're using the etcd Datastore
 *    quay.io/calico/typha:v3.4.0             # using Typha only if you're using the Kubernetes API Datastore and you have more than 50 Kubernetes nodes.
 *
 *    quay.io/calico/routereflector:v0.6.3    # Calico v3.3+ will support running calico/node in a route reflection mode
 *                                            # https://github.com/projectcalico/calico/issues/1745
 *
 *  https://github.com/kubernetes/contrib/tree/master/election
 *    gcr.io/google-containers/leader-elector:0.5
 */

package calico

const (

	Version = "v3.4.0"

	//This ConfigMap is used to configure a self-hosted Calico installation.
	//
	//https://github.com/containernetworking/plugins/tree/master/plugins/meta/bandwidth
	//https://github.com/containernetworking/plugins/tree/master/plugins/meta/tuning
	NodeConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: calico-config
  namespace: kube-system
data:
  etcd_endpoints: "https://kubernetes.cluster.local:2379"
  etcd_ca: "/etc/kubernetes/pki/etcd/ca.crt"
  etcd_cert: "/etc/kubernetes/pki/etcd/client.crt"
  etcd_key: "/etc/kubernetes/pki/etcd/client.key"
  calico_backend: "bird"
  ip: {{ .IPAutoDetection }}
  ip_autodetection_method: "first-found"
  ip6: {{ .IP6AutoDetection }}
  ip6_autodetection_method: "first-found"
  cni_network_config: |-
    {
        "name": "k8s",
        "cniVersion": "0.3.1",
        "plugins": [
          {
            "type": "calico",
            "etcd_endpoints": "__ETCD_ENDPOINTS__",
            "etcd_key_file": "__ETCD_KEY_FILE__",
            "etcd_cert_file": "__ETCD_CERT_FILE__",
            "etcd_ca_cert_file": "__ETCD_CA_CERT_FILE__",
            "log_level": "__LOG_LEVEL__",
            "mtu": 1440,
            "ipam": {
                "type": "calico-ipam",
                "assign_ipv4": "{{ .AssignIpv4 }}",
                "assign_ipv6": "{{ .AssignIpv6 }}"
             },
            "policy": {
                  "type": "k8s"
             },
            "kubernetes": {
                "kubeconfig": "/etc/kubernetes/kubelet.conf"
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

	// This manifest installs the calico/node container,
	// as well as the Calico CNI plugins and network config on
	// each master and worker node in a Kubernetes cluster.
	Node = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: calico-node
  namespace: kube-system
  labels:
    k8s-app: calico-node
spec:
  selector:
    matchLabels:
      k8s-app: calico-node
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
  template:
    metadata:
      labels:
        k8s-app: calico-node
        component: calico
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ''
    spec:
      nodeSelector:
        beta.kubernetes.io/os: linux
        beta.kubernetes.io/arch: amd64
      hostNetwork: true
      tolerations:
        - effect: NoSchedule
          operator: Exists
        - effect: NoExecute
          operator: Exists
      serviceAccountName: calico-node
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
        - name: calico-node
          image: {{ .ImageRepository }}/node:{{ .Version }}
          env:
            - name: ETCD_ENDPOINTS
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: etcd_endpoints
            - name: ETCD_CA_CERT_FILE
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: etcd_ca
            - name: ETCD_KEY_FILE
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: etcd_key
            - name: ETCD_CERT_FILE
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: etcd_cert
            - name: CALICO_NETWORKING_BACKEND
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: calico_backend
            - name: IP
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: ip
            - name: IP6
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: ip6
            - name: IP_AUTODETECTION_METHOD
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: ip_autodetection_method
            - name: IP6_AUTODETECTION_METHOD
              valueFrom:
                configMapKeyRef:
                  name: calico-config
                  key: ip6_autodetection_method
            - name: CLUSTER_TYPE
              value: "k8s,bgp"
            - name: CALICO_K8S_NODE_REF
              valueFrom:
                fieldRef:
                  fieldPath: spec.nodeName
            - name: CALICO_DISABLE_FILE_LOGGING
              value: "true"
            - name: FELIX_DEFAULTENDPOINTTOHOSTACTION
              value: "ACCEPT"
            - name: NO_DEFAULT_POOLS
              value: "true"
            - name: FELIX_IPV6SUPPORT
              value: "true"
            - name: FELIX_IPINIPMTU
              value: "1440"
            - name: FELIX_LOGSEVERITYSCREEN
              value: "info"
            - name: FELIX_HEALTHENABLED
              value: "true"
          securityContext:
            privileged: true
          resources:
            requests:
              cpu: 100m
              memory: 150Mi
            limits:
              cpu: 100m
              memory: 150Mi
          readinessProbe:
            exec:
              command:
              - /bin/calico-node
              - -bird-ready
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
        - name: k8s-certs
          hostPath:
            path: /etc/kubernetes/pki
            type: DirectoryOrCreate
        - name: etc-resolv-conf
          hostPath:
            path: /etc/resolv.conf
            type: FileOrCreate`

	// This manifest installs the calico/kube-controllers container on each master.
	// using kube-controllers only if you're using the etcd Datastore
	// See https://github.com/projectcalico/kube-controllers
	//     https://docs.projectcalico.org/v3.2/reference/kube-controllers/configuration
	//     https://github.com/kubernetes/contrib/tree/master/election

	//The calico/kube-controllers container includes the following controllers:
	//1> policy controller: watches network policies and programs Calico policies.
    //2> profile controller: watches namespaces and programs Calico profiles.
    //3> workloadendpoint controller: watches for changes to pod labels and updates Calico workload endpoints.
    //4> node controller: watches for the removal of Kubernetes nodes and removes corresponding data from Calico.
    //5> serviceAccount controller: implements the Controller interface for managing Kubernetes service account
	//   and syncing them to the Calico datastore as Profiles.
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
      name: kube-controller
      namespace: kube-system
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
        image: {{ .ImageRepository }}/kube-controllers:{{ .Version }}-fixed
        imagePullPolicy: IfNotPresent
        resources:
          requests:
            cpu: 100m
            memory: 100Mi
          limits:
            cpu: 100m
            memory: 100Mi
        env:
          - name: ETCD_ENDPOINTS
            valueFrom:
              configMapKeyRef:
                name: calico-config
                key: etcd_endpoints
          - name: ETCD_CA_CERT_FILE
            valueFrom:
              configMapKeyRef:
                name: calico-config
                key: etcd_ca
          - name: ETCD_KEY_FILE
            valueFrom:
              configMapKeyRef:
                name: calico-config
                key: etcd_key
          - name: ETCD_CERT_FILE
            valueFrom:
              configMapKeyRef:
                name: calico-config
                key: etcd_cert
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
            type: DirectoryOrCreate`

	CtlConfigMap = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Name }}
  namespace: kube-system
  labels:
    networking.projectcalico.org/name: {{ .Name }}
  annotations:
    networking.kubernetes.io/pod-cidr: {{ .PodSubnet }}
    networking.kubernetes.io/service-cidr: {{ .ServiceSubnet }}
    networking.projectcalico.org/cidr: {{ .PodSubnet }}
    networking.projectcalico.org/name: {{ .Name }}
data:
  ippool.yaml: |-
    apiVersion: projectcalico.org/v3
    kind: IPPool
    metadata:
      name: {{ .Name }}
    spec:
      cidr: {{ .PodSubnet }}
      ipipMode: Never
      natOutgoing: true
`

	CtlJob = `
apiVersion: batch/v1
kind: Job
metadata:
  labels:
    k8s-app: {{ .Name }}
  name: configure-{{ .Name }}
  namespace: kube-system
spec:
  completions: 1
  parallelism: 1
  template:
    metadata:
      labels:
        k8s-app: {{ .Name }}
    spec:
      containers:
      - args:
        - apply
        - -f
        - /etc/config/calico/ippool.yaml
        env:
        - name: ETCD_ENDPOINTS
          value: http://127.0.0.1:2379
        image: {{ .ImageRepository }}/ctl:{{ .Version }}
        imagePullPolicy: IfNotPresent
        name: configure-calico
        volumeMounts:
        - mountPath: /etc/config
          name: config-volume
      hostNetwork: true
      nodeSelector:
        node-role.kubernetes.io/master: ""
      tolerations:
        - effect: NoSchedule
          operator: Exists
        - effect: NoExecute
          operator: Exists
      restartPolicy: OnFailure
      volumes:
      - configMap:
          defaultMode: 420
          items:
          - key: ippool.yaml
            path: calico/ippool.yaml
          name: {{ .Name }}
        name: config-volume
`
	// for calico/node
	CalicoClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:calico-node
rules:
  - apiGroups: [""]
    resources:
      - pods
      - nodes
      - namespaces
    verbs:
      - get
`

	CalicoServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: calico-node
  namespace: kube-system`

	CalicoClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:calico-node
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:calico-node
subjects:
- kind: ServiceAccount
  name: calico-node
  namespace: kube-system`

	// for calico/kube-controllers
	CalicoControllersClusterRole = `
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: system:kube-controllers
rules:
  - apiGroups:
    - ""
    - extensions
    resources:
      - pods
      - namespaces
      - networkpolicies
      - nodes
      - serviceaccounts
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

	CalicoControllersServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-controllers
  namespace: kube-system`

	CalicoControllersClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:kube-controllers
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:kube-controllers
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: system:masters
- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: system:nodes
- kind: ServiceAccount
  name: kube-controllers
  namespace: kube-system`
)
