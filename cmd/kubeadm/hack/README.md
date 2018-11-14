
# 制作镜像的说明

```
${REGISTRY_SERVER}/${REGISTRY_USER}/agent:${AGENT_VERSION}
${REGISTRY_SERVER}/${REGISTRY_USER}/kubectl-amd64:${K8S_VERSION}
${REGISTRY_SERVER}/${REGISTRY_USER}/tde:v3.0.0
```

## hyperkube

特别需要注意的地方是${REGISTRY_SERVER}/${REGISTRY_USER}/hyperkube:${K8S_VERSION}的tag要和
gcr.io/google_containers/hyperkube:${VERSION}的tag保持一致


请使用root用户编译hyperkube
```
root@k8s:~# git clone http://gitlab.tenxcloud.com/enterprise-2.0/contrib -b dev-branch
root@k8s:~# git clone git@gitlab.tenxcloud.com:kubernetes/kubernetes.git -b release-1.6.11
root@k8s:~# cd ${KUBE_ROOT}

root@k8s:~# 修改hack/lib/version.sh中66行代码如下
            if [[ -n ${KUBE_GIT_VERSION-} ]] || KUBE_GIT_VERSION=$("${git[@]}" describe --tags --abbrev=0 "${KUBE_GIT_COMMIT}^{commit}" 2>/dev/null); then
            
root@k8s:~# make clean && make all WHAT=cmd/hyperkube GOFLAGS=-v
root@k8s:~# cp _output/bin/hyperkube  ${HOME}/contrib/Images/hyperkube/debian
root@k8s:~# cd ${HOME}/contrib/Images/hyperkube/debian
root@k8s:~# make build
root@k8s:~# docker push 192.168.1.55/tenx_containers/hyperkube-amd64:${K8S_VERSION}

```



## kubectl-amd64

${REGISTRY_SERVER}/${REGISTRY_USER}/kubectl-amd64:${K8S_VERSION}镜像中使用的
kubectl的版本要和上述hyperkube的版本保持一致，并且其中的docker版本要和hyperkube的版本兼容

具体参考http://gitlab.tenxcloud.com/enterprise-2.0/contrib/tree/dev-branch/Images/webterminal

kubernetes 与docker 版本的兼容性参考一下文章
https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG-1.9.md#external-dependencies
https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG.md#external-dependency-version-information
http://wiki.tenxcloud.com/pages/viewpage.action?pageId=9077545

## agent
hyperkube >=v1.9时agent版本要求使用agent >=3.0


## calico 

[calico](https://docs.projectcalico.org/v2.6/releases)    v2.6.7  30 January 2018

|Component	            | Version|
| :------               | :------:|
|felix	                | [2.6.6](https://github.com/projectcalico/felix/releases/tag/2.6.6)   |
|typha	                | [v0.5.6](https://github.com/projectcalico/typha/releases/tag/v0.5.6)  |
|calicoctl	            | [v1.6.3](https://github.com/projectcalico/calicoctl/releases/tag/v1.6.3)  |
|calico/node            | [v2.6.7](https://github.com/projectcalico/calico/releases/tag/v2.6.7)  |
|calico/cni	            | [v1.11.2](https://github.com/projectcalico/cni-plugin/releases/tag/v1.11.2)  |
|confd                  | [v0.12.1-calico-0.4.3](https://github.com/projectcalico/confd/releases/tag/v0.12.1-calico-0.4.3) |
|libnetwork-plugin      | [v1.1.2](https://github.com/projectcalico/libnetwork-plugin/releases/tag/v1.1.2) |
|calico/kube-controller | [v1.0.3](https://github.com/projectcalico/kube-controllers/releases/tag/v1.0.3) |
|calico-bird            | [v0.3.2](https://github.com/projectcalico/bird/releases/tag/v0.3.2) |
|calico-bgp-daemon      | [v0.2.2](https://github.com/projectcalico/calico-bgp-daemon/releases/tag/v0.2.2) |
|networking-calico      | 1.4.3  |
|calico/routereflector  | v0.4.2 |


## TenxCloud Deployment Engine(TDE)

请使用root用户编译kubeadm

```
root@k8s:~# git clone git@gitlab.tenxcloud.com:kubernetes/kubernetes.git -b release-1.9.2
root@k8s:~# cd ${KUBE_ROOT}
root@k8s:~# make clean && make all WHAT=cmd/kubeadm GOFLAGS=-v
root@k8s:~# cp _output/bin/kubeadm  cmd/kubeadm/build/
root@k8s:~# cd  cmd/kubeadm/build/
root@k8s:~# make tde
root@k8s:~# docker push 192.168.1.55/tenx_containers/tde:v3.0.0 

```








