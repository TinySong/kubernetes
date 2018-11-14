/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2019 TenxCloud. All Rights Reserved.
 * 2019-04-12  @author weiwei@tenxcloud.com
 */

package ovn


const (

   ConfigMap = `
kind: ConfigMap
apiVersion: v1
metadata:
  name: ovn-config
  namespace: ovn-kubernetes
data:
  net_cidr:      {{ .PodSubnet }}/26
  svc_cidr:      {{ .ServiceSubnet }}
  k8s_apiserver: {{ .APIEndpoint }}
`


   DBDeploy = `
kind: Deployment
apiVersion: apps/v1
metadata:
  name: ovnkube-db
  namespace: ovn-kubernetes
  annotations:
    kubernetes.io/description: |
      This daemonset launches the OVN NB/SB ovsdb service components.
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      name: ovnkube-db
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      labels:
        name: ovnkube-db
        component: ovn
        type: infra
        beta.kubernetes.io/os: "linux"
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ''
    spec:
      serviceAccountName: ovn
      hostNetwork: true
      hostPID: true
      containers:
      - name: nb-ovsdb
        image: {{ .ImageRepository }}/ovn:latest
        imagePullPolicy: IfNotPresent
        command: ["/root/ovnkube.sh", "nb-ovsdb"]
        securityContext:
          runAsUser: 0
          privileged: true
        volumeMounts:
        - mountPath: /etc/sysconfig/origin-node
          name: host-sysconfig-node
          readOnly: true
        - mountPath: /var/run
          name: host-var-run
        - mountPath: /var/run/dbus/
          name: host-var-run-dbus
          readOnly: true
        - mountPath: /etc/openvswitch/
          name: host-var-lib-ovs
        - mountPath: /var/lib/openvswitch/
          name: host-var-lib-ovs
        - mountPath: /var/log/openvswitch/
          name: host-var-log-ovs
        - mountPath: /var/run/openvswitch/
          name: host-var-run-ovs
        resources:
          requests:
            cpu: 100m
            memory: 300Mi
        env:
        - name: OVN_DAEMONSET_VERSION
          value: "3"
        - name: OVN_LOG_NB
          value: "-vconsole:info -vfile:info"
        - name: K8S_APISERVER
          valueFrom:
            configMapKeyRef:
              name: ovn-config
              key: k8s_apiserver
        - name: K8S_NODE
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        ports:
        - name: healthz
          containerPort: 10256
      - name: sb-ovsdb
        image: {{ .ImageRepository }}/ovn:latest
        imagePullPolicy: IfNotPresent
        command: ["/root/ovnkube.sh", "sb-ovsdb"]
        securityContext:
          runAsUser: 0
          privileged: true
        volumeMounts:
        - mountPath: /etc/sysconfig/origin-node
          name: host-sysconfig-node
          readOnly: true
        - mountPath: /var/run
          name: host-var-run
        - mountPath: /var/run/dbus/
          name: host-var-run-dbus
          readOnly: true
        - mountPath: /etc/openvswitch/
          name: host-var-lib-ovs
        - mountPath: /var/lib/openvswitch/
          name: host-var-lib-ovs
        - mountPath: /var/log/openvswitch/
          name: host-var-log-ovs
        - mountPath: /var/run/openvswitch/
          name: host-var-run-ovs
        resources:
          requests:
            cpu: 100m
            memory: 300Mi
        env:
        - name: OVN_DAEMONSET_VERSION
          value: "3"
        - name: OVN_LOG_SB
          value: "-vconsole:info -vfile:info"
        - name: K8S_APISERVER
          valueFrom:
            configMapKeyRef:
              name: ovn-config
              key: k8s_apiserver
        - name: K8S_NODE
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        ports:
        - name: healthz
          containerPort: 10255
      nodeSelector:
        node-role.kubernetes.io/master: ""
        beta.kubernetes.io/os: "linux"
      volumes:
      - name: host-sysconfig-node
        hostPath:
          path: /etc/sysconfig/origin-node
      - name: host-modules
        hostPath:
          path: /lib/modules
      - name: host-var-run
        hostPath:
          path: /var/run
      - name: host-var-run-dbus
        hostPath:
          path: /var/run/dbus
      - name: host-var-lib-ovs
        hostPath:
          path: /var/lib/openvswitch
      - name: host-var-log-ovs
        hostPath:
          path: /var/log/openvswitch
      - name: host-var-run-ovs
        hostPath:
          path: /var/run/openvswitch
      tolerations:
      - operator: "Exists"
`

   DBService = `
apiVersion: v1
kind: Service
metadata:
  name: ovnkube-db
  namespace: ovn-kubernetes
spec:
  ports:
  - name: north
    port: 6641
    protocol: TCP
    targetPort: 6641
  - name: south
    port: 6642
    protocol: TCP
    targetPort: 6642
  sessionAffinity: None
  clusterIP: None
  type: ClusterIP
`


   NorthdDeploy = `
kind: Deployment
apiVersion: apps/v1
metadata:
  name: ovnkube-master
  namespace: ovn-kubernetes
  annotations:
    kubernetes.io/description: |
      This daemonset launches the ovn-kubernetes networking components.
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      name: ovnkube-master
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      labels:
        name: ovnkube-master
        component: ovn
        type: infra
        openshift.io/component: network
        beta.kubernetes.io/os: "linux"
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ''
    spec:
      serviceAccountName: ovn
      hostNetwork: true
      hostPID: true
      affinity:
        podAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
                - key: name
                  operator: In
                  values:
                    - ovnkube-db
            topologyKey: kubernetes.io/hostname
      containers:
      - name: run-ovn-northd
        image: {{ .ImageRepository }}/ovn:latest
        imagePullPolicy: IfNotPresent
        command: ["/root/ovnkube.sh", "run-ovn-northd"]
        securityContext:
          runAsUser: 0
          privileged: true
        volumeMounts:
        - mountPath: /etc/sysconfig/origin-node
          name: host-sysconfig-node
          readOnly: true
        - mountPath: /var/run
          name: host-var-run
        - mountPath: /var/run/dbus/
          name: host-var-run-dbus
          readOnly: true
        - mountPath: /etc/openvswitch/
          name: host-var-lib-ovs
        - mountPath: /var/lib/openvswitch/
          name: host-var-lib-ovs
        - mountPath: /var/log/openvswitch/
          name: host-var-log-ovs
        - mountPath: /var/run/openvswitch/
          name: host-var-run-ovs
        - mountPath: /var/run/kubernetes/
          name: host-var-run-kubernetes
          readOnly: true
        - mountPath: /var/run/ovn-kubernetes
          name: host-var-run-ovn-kubernetes
        - mountPath: /host/opt/cni/bin
          name: host-opt-cni-bin
        - mountPath: /etc/cni/net.d
          name: host-etc-cni-netd
        - mountPath: /var/lib/cni/networks/ovn-k8s-cni-overlay
          name: host-var-lib-cni-networks-ovn-kubernetes
        resources:
          requests:
            cpu: 100m
            memory: 300Mi
        env:
        - name: OVN_DAEMONSET_VERSION
          value: "3"
        - name: OVN_LOG_NORTHD
          value: "-vconsole:info"
        - name: OVN_NET_CIDR
          valueFrom:
            configMapKeyRef:
              name: ovn-config
              key: net_cidr
        - name: OVN_SVC_CIDR
          valueFrom:
            configMapKeyRef:
              name: ovn-config
              key: svc_cidr
        - name: K8S_APISERVER
          valueFrom:
            configMapKeyRef:
              name: ovn-config
              key: k8s_apiserver
        - name: K8S_NODE
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        ports:
        - name: healthz
          containerPort: 10257
      - name: ovnkube-master
        image: {{ .ImageRepository }}/ovn:latest
        imagePullPolicy: IfNotPresent
        command: ["/root/ovnkube.sh", "ovn-master"]
        securityContext:
          runAsUser: 0
          privileged: true
        volumeMounts:
        - mountPath: /etc/sysconfig/origin-node
          name: host-sysconfig-node
          readOnly: true
        - mountPath: /var/run
          name: host-var-run
        - mountPath: /var/run/dbus/
          name: host-var-run-dbus
          readOnly: true
        - mountPath: /etc/openvswitch/
          name: host-var-lib-ovs
        - mountPath: /var/lib/openvswitch/
          name: host-var-lib-ovs
        - mountPath: /var/log/openvswitch/
          name: host-var-log-ovs
        - mountPath: /var/log/ovn-kubernetes/
          name: host-var-log-ovnkube
        - mountPath: /var/run/openvswitch/
          name: host-var-run-ovs
        - mountPath: /var/run/kubernetes/
          name: host-var-run-kubernetes
          readOnly: true
        - mountPath: /var/run/ovn-kubernetes
          name: host-var-run-ovn-kubernetes
        - mountPath: /host/opt/cni/bin
          name: host-opt-cni-bin
        - mountPath: /etc/cni/net.d
          name: host-etc-cni-netd
        - mountPath: /var/lib/cni/networks/ovn-k8s-cni-overlay
          name: host-var-lib-cni-networks-ovn-kubernetes
        resources:
          requests:
            cpu: 100m
            memory: 300Mi
        env:
        - name: OVN_DAEMONSET_VERSION
          value: "3"
        - name: OVN_MASTER
          value: "true"
        - name: OVNKUBE_LOGLEVEL
          value: "4"
        - name: OVN_NET_CIDR
          valueFrom:
            configMapKeyRef:
              name: ovn-config
              key: net_cidr
        - name: OVN_SVC_CIDR
          valueFrom:
            configMapKeyRef:
              name: ovn-config
              key: svc_cidr
        - name: K8S_APISERVER
          valueFrom:
            configMapKeyRef:
              name: ovn-config
              key: k8s_apiserver
        - name: K8S_NODE
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        ports:
        - name: healthz
          containerPort: 10254
      nodeSelector:
        node-role.kubernetes.io/master: ""
        beta.kubernetes.io/os: "linux"
      volumes:
      - name: host-sysconfig-node
        hostPath:
          path: /etc/sysconfig/origin-node
      - name: host-modules
        hostPath:
          path: /lib/modules
      - name: host-var-run
        hostPath:
          path: /var/run
      - name: host-var-run-dbus
        hostPath:
          path: /var/run/dbus
      - name: host-var-lib-ovs
        hostPath:
          path: /var/lib/openvswitch
      - name: host-var-log-ovs
        hostPath:
          path: /var/log/openvswitch
      - name: host-var-log-ovnkube
        hostPath:
          path: /var/log/ovn-kubernetes
      - name: host-var-run-ovs
        hostPath:
          path: /var/run/openvswitch
      - name: host-var-run-kubernetes
        hostPath:
          path: /var/run/kubernetes
      - name: host-var-run-ovn-kubernetes
        hostPath:
          path: /var/run/ovn-kubernetes
      - name: host-opt-cni-bin
        hostPath:
          path: /opt/cni/bin
      - name: host-etc-cni-netd
        hostPath:
          path: /etc/cni/net.d
      - name: host-var-lib-cni-networks-ovn-kubernetes
        hostPath:
          path: /var/lib/cni/networks/ovn-k8s-cni-overlay
      tolerations:
      - operator: "Exists"
`


   Controller = `
kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: ovnkube-node
  namespace: ovn-kubernetes
  annotations:
    kubernetes.io/description: |
      This daemonset launches the ovn-kubernetes networking components.
spec:
  selector:
    matchLabels:
      app: ovnkube-node
  updateStrategy:
    type: RollingUpdate
  template:
    metadata:
      labels:
        app: ovnkube-node
        component: ovn
        type: infra
        beta.kubernetes.io/os: "linux"
      annotations:
        scheduler.alpha.kubernetes.io/critical-pod: ''
    spec:
      serviceAccountName: ovn
      hostNetwork: true
      hostPID: true
      containers:
      - name: ovs-daemons
        image: {{ .ImageRepository }}/ovn:latest
        imagePullPolicy: IfNotPresent
        command: ["/root/ovnkube.sh", "ovs-server"]
        livenessProbe:
          exec:
            command:
            - /usr/share/openvswitch/scripts/ovs-ctl
            - status
          initialDelaySeconds: 15
          periodSeconds: 5
        securityContext:
          runAsUser: 0
          privileged: true
        volumeMounts:
        - mountPath: /lib/modules
          name: host-modules
          readOnly: true
        - mountPath: /run/openvswitch
          name: host-run-ovs
        - mountPath: /var/run/openvswitch
          name: host-var-run-ovs
        - mountPath: /sys
          name: host-sys
          readOnly: true
        - mountPath: /etc/openvswitch
          name: host-config-openvswitch
        resources:
          requests:
            cpu: 100m
            memory: 300Mi
          limits:
            cpu: 200m
            memory: 400Mi
        env:
        - name: OVN_DAEMONSET_VERSION
          value: "3"
        - name: K8S_APISERVER
          valueFrom:
            configMapKeyRef:
              name: ovn-config
              key: k8s_apiserver
      - name: ovn-controller
        image: {{ .ImageRepository }}/ovn:latest
        imagePullPolicy: IfNotPresent
        command: ["/root/ovnkube.sh", "ovn-controller"]
        securityContext:
          runAsUser: 0
          privileged: true
        volumeMounts:
        - mountPath: /etc/sysconfig/origin-node
          name: host-sysconfig-node
          readOnly: true
        - mountPath: /var/run
          name: host-var-run
        - mountPath: /var/run/dbus/
          name: host-var-run-dbus
          readOnly: true
        - mountPath: /var/lib/openvswitch/
          name: host-var-lib-ovs
        - mountPath: /var/log/openvswitch/
          name: host-var-log-ovs
        - mountPath: /var/run/openvswitch/
          name: host-var-run-ovs
        - mountPath: /var/run/kubernetes/
          name: host-var-run-kubernetes
          readOnly: true
        - mountPath: /var/run/ovn-kubernetes
          name: host-var-run-ovn-kubernetes
        - mountPath: /host/opt/cni/bin
          name: host-opt-cni-bin
        - mountPath: /etc/cni/net.d
          name: host-etc-cni-netd
        - mountPath: /var/lib/cni/networks/ovn-k8s-cni-overlay
          name: host-var-lib-cni-networks-ovn-kubernetes
        resources:
          requests:
            cpu: 100m
            memory: 300Mi
        env:
        - name: OVN_DAEMONSET_VERSION
          value: "3"
        - name: OVNKUBE_LOGLEVEL
          value: "4"
        - name: OVN_NET_CIDR
          valueFrom:
            configMapKeyRef:
              name: ovn-config
              key: net_cidr
        - name: OVN_SVC_CIDR
          valueFrom:
            configMapKeyRef:
              name: ovn-config
              key: svc_cidr
        - name: K8S_APISERVER
          valueFrom:
            configMapKeyRef:
              name: ovn-config
              key: k8s_apiserver
        - name: K8S_NODE
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        ports:
        - name: healthz
          containerPort: 10258
      - name: ovn-node
        image: {{ .ImageRepository }}/ovn:latest
        imagePullPolicy: IfNotPresent
        command: ["/root/ovnkube.sh", "ovn-node"]
        securityContext:
          runAsUser: 0
          privileged: true
        volumeMounts:
        - mountPath: /etc/sysconfig/origin-node
          name: host-sysconfig-node
          readOnly: true
        - mountPath: /var/run
          name: host-var-run
        - mountPath: /var/run/dbus/
          name: host-var-run-dbus
          readOnly: true
        - mountPath: /var/lib/openvswitch/
          name: host-var-lib-ovs
        - mountPath: /var/log/openvswitch/
          name: host-var-log-ovs
        - mountPath: /var/log/ovn-kubernetes/
          name: host-var-log-ovnkube
        - mountPath: /var/run/openvswitch/
          name: host-var-run-ovs
        - mountPath: /var/run/kubernetes/
          name: host-var-run-kubernetes
          readOnly: true
        - mountPath: /var/run/ovn-kubernetes
          name: host-var-run-ovn-kubernetes
        - mountPath: /host/opt/cni/bin
          name: host-opt-cni-bin
        - mountPath: /etc/cni/net.d
          name: host-etc-cni-netd
        - mountPath: /var/lib/cni/networks/ovn-k8s-cni-overlay
          name: host-var-lib-cni-networks-ovn-kubernetes
        resources:
          requests:
            cpu: 100m
            memory: 300Mi
        env:
        - name: OVN_DAEMONSET_VERSION
          value: "3"
        - name: OVNKUBE_LOGLEVEL
          value: "5"
        - name: OVN_NET_CIDR
          valueFrom:
            configMapKeyRef:
              name: ovn-config
              key: net_cidr
        - name: OVN_SVC_CIDR
          valueFrom:
            configMapKeyRef:
              name: ovn-config
              key: svc_cidr
        - name: K8S_APISERVER
          valueFrom:
            configMapKeyRef:
              name: ovn-config
              key: k8s_apiserver
        - name: K8S_NODE
          valueFrom:
            fieldRef:
              fieldPath: spec.nodeName
        ports:
        - name: healthz
          containerPort: 10259
      nodeSelector:
        beta.kubernetes.io/os: "linux"
      volumes:
      - name: host-sysconfig-node
        hostPath:
          path: /etc/sysconfig/origin-node
      - name: host-modules
        hostPath:
          path: /lib/modules
      - name: host-var-run
        hostPath:
          path: /var/run
      - name: host-var-run-dbus
        hostPath:
          path: /var/run/dbus
      - name: host-var-lib-ovs
        hostPath:
          path: /var/lib/openvswitch
      - name: host-var-log-ovs
        hostPath:
          path: /var/log/openvswitch
      - name: host-var-log-ovnkube
        hostPath:
          path: /var/log/ovn-kubernetes
      - name: host-run-ovs
        hostPath:
          path: /run/openvswitch
      - name: host-var-run-ovs
        hostPath:
          path: /var/run/openvswitch
      - name: host-var-run-kubernetes
        hostPath:
          path: /var/run/kubernetes
      - name: host-var-run-ovn-kubernetes
        hostPath:
          path: /var/run/ovn-kubernetes
      - name: host-sys
        hostPath:
          path: /sys
      - name: host-opt-cni-bin
        hostPath:
          path: /opt/cni/bin
      - name: host-etc-cni-netd
        hostPath:
          path: /etc/cni/net.d
      - name: host-config-openvswitch
        hostPath:
          path: /etc/origin/openvswitch
      - name: host-var-lib-cni-networks-ovn-kubernetes
        hostPath:
          path: /var/lib/cni/networks/ovn-k8s-cni-overlay
      tolerations:
      - operator: "Exists"
`


   ServiceAccount = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: ovn
  namespace: ovn-kubernetes
`

   ClusterRole = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  annotations:
    rbac.authorization.k8s.io/system-only: "true"
  name: system:ovn
rules:
- apiGroups:
  - ""
  - extensions
  resources:
  - pods
  - namespaces
  - networkpolicies
  - nodes
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - networking.k8s.io
  resources:
  - networkpolicies
  verbs:
  - get
  - list
  - watch
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create
  - patch
  - update
- apiGroups:
  - policy
  resources: 
  - podsecuritypolicies
  verbs:     
  - use
  resourceNames:
  - system
- apiGroups:
  - extensions
  resources: 
  - podsecuritypolicies
  verbs:     
  - use
  resourceNames:
  - system
`

   ClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:ovn
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:ovn
subjects:
- kind: ServiceAccount
  name: ovn
  namespace: ovn-kubernetes
`

   AdminClusterRoleBinding = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: system:ovn-admin
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: ovn
  namespace: ovn-kubernetes
`


)

