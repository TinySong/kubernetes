/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-10-10  @author weiwei@tenxcloud.com
 */
package kubediscovery


const (

	Version         = "v4.0.0"

	KubeDiscovery = `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  labels:
    name: kube-discovery
  name: kube-discovery
  namespace: kube-system
spec:
  selector:
    matchLabels:
      name: kube-discovery
  template:
    metadata:
      labels:
        name: kube-discovery
    spec:
      containers:
      - name: discovery
        image: {{ .ImageRepository }}/kube-discovery-{{ .Arch }}:{{ .Version }}
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 9898
          protocol: TCP
        livenessProbe:
          httpGet:
            path: /liveness
            port: 9898
            scheme: HTTP
          failureThreshold: 8
          initialDelaySeconds: 60
          timeoutSeconds: 15
        volumeMounts:
        - name: pki
          mountPath: /tmp/secret
          readOnly: false
      dnsPolicy: ClusterFirst
      hostNetwork: true
      nodeSelector:
        node-role.kubernetes.io/master: ""
      restartPolicy: Always
      tolerations:
      - effect: NoSchedule
        operator: Exists
      - effect: NoExecute
        operator: Exists
      volumes:
      - name: pki
        secret:
          secretName: {{ .SecretName }}
`

)