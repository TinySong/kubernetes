/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-01-23  @author weiwei@tenxcloud.com
 */
package addons

import (
	"fmt"

	clientset "k8s.io/client-go/kubernetes"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"

	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/network"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/dns"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/dnsautoscaler"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/kubediscovery"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/kubectl"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/proxy"
	"k8s.io/kubernetes/cmd/kubeadm/app/phases/addons/serviceproxy"
)

// deploy all addons
// kube-dns,kube-proxy,kube-discovery
// network plugin(calico,flannel,canal,macvlan),service-proxy
// kubectl
func DeployAddons(cfg *kubeadmapi.InitConfiguration, client clientset.Interface) error {

	if err := kubediscovery.CreateKubeDiscoveryAddon(cfg,client); err != nil {
		return fmt.Errorf("error setup kube-discovery addon: %v", err)
	}

	if err := network.DeployNetworkAddons(cfg,client); err != nil {
		return fmt.Errorf("error setup network addon: %v", err)
	}

	if err := dns.EnsureDNSAddon(cfg, client); err != nil {
		return fmt.Errorf("error setup kube-dns addon: %v", err)
	}

	if err := dnsautoscaler.CreateDnsAutoscalerAddOns(cfg, client); err != nil {
		return fmt.Errorf("error setup dns-autoscaler addon: %v", err)
	}

	if err := proxy.EnsureProxyAddon(cfg, client); err != nil {
		return fmt.Errorf("error setup kube-proxy addon: %v", err)
	}

	if err := kubectl.CreateKubectlAddon(cfg, client); err != nil {
		return fmt.Errorf("error setup kubectl addon: %v", err)
	}

	if err := serviceproxy.CreateTenxProxyAddon(cfg, client); err != nil {
		return fmt.Errorf("error setup service-proxy addon: %v", err)
	}

	return nil
}

