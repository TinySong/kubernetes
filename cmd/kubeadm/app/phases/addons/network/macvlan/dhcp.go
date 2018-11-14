/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-10-10  @author weiwei@tenxcloud.com
 */
package macvlan

import (
	"bytes"
	"fmt"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/util/initsystem"
)

var (
	dhcpService            = "/etc/systemd/system/dhcp.service"
	ServiceName            = "dhcp"
)

func TrySetupDHCP() error {
	// PHASE 1: Write DHCP Proxy Service to /etc/systemd/system/kubelet.service
	err := writeDHCPService()
	if err != nil {
		fmt.Println("[dhcp] Write dhcp service to /etc/systemd/system/dhcp.service failed.")
		return err
	}
	// PHASE 2: If we notice that the dhcp service is inactive, try to start it
	initSystem, err := initsystem.GetInitSystem()
	if err != nil {
		fmt.Println("[dhcp] No supported init system detected, won't ensure dhcp is running.")
		return err
	} else if initSystem.ServiceExists(ServiceName) && !initSystem.ServiceIsActive(ServiceName) {
		fmt.Println("[dhcp] Starting the dhcp service")
		if err := initSystem.ServiceStart(ServiceName); err != nil {
			fmt.Printf("[dhcp] WARNING: Unable to start the dhcp service: [%v]\n", err)
			fmt.Println("[dhcp] WARNING: Please ensure dhcp is running manually.")
			return err
		} else {
			if !initSystem.ServiceIsEnabled(ServiceName) {
				initSystem.ServiceEnable(ServiceName)
				fmt.Println("[dhcp] dhcp proxy is enabled.")
			}
		}
	}
	return nil
}


// /etc/systemd/system/dhcp.service
func writeDHCPService() error {
	buf := bytes.Buffer{}
	buf.WriteString("[Unit] \n")
	buf.WriteString("Description=dhcp: The cni dhcp proxy \n")
	buf.WriteString("Documentation=https://github.com/containernetworking/plugins \n")
	buf.WriteString("After=network.target \n")
	buf.WriteString(" \n")
	buf.WriteString("[Service] \n")
	buf.WriteString("ExecStartPre=/bin/bash -c \"rm -f /run/cni/dhcp.sock\" \n")
	buf.WriteString("ExecStart=/opt/cni/bin/dhcp daemon \n")
	buf.WriteString("ExecStopPost=/bin/bash -c \"rm -f /run/cni/dhcp.sock\" \n")
	buf.WriteString("Restart=on-failure \n")
	buf.WriteString("StartLimitInterval=0 \n")
	buf.WriteString("RestartSec=10 \n")
	buf.WriteString(" \n")
	buf.WriteString("[Install] \n")
	buf.WriteString("WantedBy=multi-user.target \n")
	if err := cmdutil.DumpReaderToFile(bytes.NewReader(buf.Bytes()), dhcpService); err != nil {
		return fmt.Errorf("failed to create dhcp.service file for [%v] \n", err)
	} else {
		fmt.Printf("[dhcp] Write dhcp service to %q Successfully.\n", dhcpService)
	}
	return nil
}