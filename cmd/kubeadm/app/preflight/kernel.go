/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-10-10  @author weiwei@tenxcloud.com
 */
package preflight

import (
	"os"
	"fmt"
	"os/exec"
	"bufio"
)

const (
	kernelModuleFile string = "/etc/modules-load.d/k8s.conf"
	sysctlFile       string = "/etc/sysctl.d/k8s-sysctl.conf"
)

var kernelModules = []string{
	"br_netfilter",
	"ip_vs",
	"ip_vs_wrr",
	"ip_vs_sh",
	"ip_vs_rr",
	"nf_conntrack_ipv4",
}

var sysctl  = []string {
	"net.ipv4.ip_forward=1",
	"net.ipv6.conf.all.forwarding=1",
	"net.bridge.bridge-nf-call-iptables=1",
	"net.bridge.bridge-nf-call-ip6tables=1",
}

// RunInitMasterChecks
// RunJoinNodeChecks
func LoadKernelModule() {
	insertKernelModule()
	setupKernelSysctl()
}

func insertKernelModule() {
	//Step 1: load kernel module into configuration
	writeLines(kernelModules, kernelModuleFile)
	//Step 2: insert kernel module into kernel
	var modules string
	for _, module := range kernelModules {
		modules = modules + fmt.Sprintf("%s ", module)
	}
	cmd := fmt.Sprintf("modprobe -a %s",modules)
	//fmt.Printf("[preflight] load kernel module [%s]\n",modules)
	execCmd(cmd)
}

func setupKernelSysctl() {
	//Step 1: load kernel params into configuration
	writeLines(sysctl, sysctlFile)
	//Step 2: insert kernel params into kernel
	cmd := fmt.Sprintf("sysctl --system")
	//fmt.Printf("SetupKernelSysctl [%s]",cmd)
	execCmd(cmd)
}



func execCmd(cmd string) {
	_, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		fmt.Errorf("failed to exec cmd [%s] : %v\n", cmd, err)
	}
}

func writeLines(lines []string, path string) error {
	deleteFile(path)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}

func deleteFile(file string) {
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		if err := os.Remove(file); err != nil {
			fmt.Errorf("failed to remove file [%s]",file)
		}
	}
}
