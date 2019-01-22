/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package etcd

import (
	"fmt"
	"path"
	"strings"
	"context"
	"path/filepath"

	"k8s.io/api/core/v1"
	"github.com/golang/glog"
	"github.com/coreos/etcd/clientv3"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	"k8s.io/kubernetes/cmd/kubeadm/app/images"
	kubeadmutil "k8s.io/kubernetes/cmd/kubeadm/app/util"
	etcdClient "k8s.io/kubernetes/cmd/kubeadm/app/util/etcd"
	staticpodutil "k8s.io/kubernetes/cmd/kubeadm/app/util/staticpod"

	"time"
)

const (
	etcdVolumeName      = "etcd-data"
	certsVolumeName     = "etcd-certs"
	localTimeVolumeName = "localtime"
	localTimeVolumePath = "/etc/localtime"
)

// CreateLocalEtcdStaticPodManifestFile will write local etcd static pod manifest file.
func CreateLocalEtcdStaticPodManifestFile(manifestDir string, cfg *kubeadmapi.InitConfiguration) error {
	glog.V(1).Infoln("creating local etcd static pod manifest file")
	// gets etcd StaticPodSpec, actualized for the current InitConfiguration
	spec := GetEtcdPodSpec(cfg)
	// writes etcd StaticPod to disk
	if err := staticpodutil.WriteStaticPodToDisk(kubeadmconstants.Etcd, manifestDir, spec); err != nil {
		return err
	}

	fmt.Printf("[etcd] Wrote Static Pod manifest for a local etcd instance to %q\n", kubeadmconstants.GetStaticPodFilepath(kubeadmconstants.Etcd, manifestDir))
	return nil
}

// GetEtcdPodSpec returns the etcd static Pod actualized to the context of the current InitConfiguration
// NB. GetEtcdPodSpec methods holds the information about how kubeadm creates etcd static pod manifests.
func GetEtcdPodSpec(cfg *kubeadmapi.InitConfiguration) v1.Pod {
	pathType := v1.HostPathDirectoryOrCreate
	hostPathFileOrCreate := v1.HostPathFileOrCreate
	etcdMounts := map[string]v1.Volume{
		etcdVolumeName:      staticpodutil.NewVolume(etcdVolumeName, cfg.Etcd.Local.DataDir, &pathType),
		certsVolumeName:     staticpodutil.NewVolume(certsVolumeName, cfg.CertificatesDir+"/etcd", &pathType),
		localTimeVolumeName: staticpodutil.NewVolume(localTimeVolumeName, localTimeVolumePath, &hostPathFileOrCreate),
	}
	return staticpodutil.ComponentPod(v1.Container{
		Name:            kubeadmconstants.Etcd,
		Command:         getEtcdCommand(cfg),
		Image:           images.GetEtcdImage(&cfg.ClusterConfiguration),
		ImagePullPolicy: v1.PullIfNotPresent,
		// Mount the etcd datadir path read-write so etcd can store data in a more persistent manner
		VolumeMounts: []v1.VolumeMount{
			staticpodutil.NewVolumeMount(etcdVolumeName, cfg.Etcd.Local.DataDir, false),
			staticpodutil.NewVolumeMount(certsVolumeName, cfg.CertificatesDir+"/etcd", false),
			staticpodutil.NewVolumeMount(localTimeVolumeName, localTimeVolumePath, true),
		},
		LivenessProbe: staticpodutil.ComponentProbe(cfg, kubeadmconstants.Etcd, 2379, "/health", v1.URISchemeHTTP),
		//LivenessProbe: staticpodutil.EtcdProbe(
		//	cfg, kubeadmconstants.Etcd, 2379, cfg.CertificatesDir,
		//	kubeadmconstants.EtcdCACertName, kubeadmconstants.EtcdHealthcheckClientCertName, kubeadmconstants.EtcdHealthcheckClientKeyName,
		//),
	}, etcdMounts)
}

// getEtcdCommand builds the right etcd command from the given config object
func getEtcdCommand(cfg *kubeadmapi.InitConfiguration) []string {
	var loopback string
	if cfg.Networking.Mode == kubeadmconstants.NetworkIPV6Mode || cfg.Networking.Mode == kubeadmconstants.NetworkDualStackMode {
		loopback = "[::1]"
	} else {
		loopback = "127.0.0.1"
	}
	advertiseAddr := loopback
	if len(cfg.APIEndpoint.AdvertiseAddress) > 0 {
		advertiseAddr = cfg.APIEndpoint.AdvertiseAddress
	}
	newMemberPeerUrl := "https://" + advertiseAddr + ":2380"
	initialCluster := cfg.GetNodeName() + "=https://" + advertiseAddr + ":2380"
	initialClusterState := "new"

	if len(cfg.DiscoveryTokenAPIServers) != 0 {
		err := wait.PollImmediateInfinite(kubeadmconstants.DiscoveryRetryInterval, func() (bool, error) {
			apiServers := cfg.DiscoveryTokenAPIServers[0]
			socket := strings.Split(apiServers, ":")
			existingMember := fmt.Sprintf("https://%s:2379", socket[0])
			fmt.Printf("[etcd] Adding etcd member [ %s ] into an existing cluster [ %s ] \n", newMemberPeerUrl, existingMember)
			client, err := etcdClient.NewEtcdClient([]string{existingMember},
				path.Join(cfg.CertificatesDir, kubeadmconstants.APIServerEtcdClientCertName),
				path.Join(cfg.CertificatesDir, kubeadmconstants.APIServerEtcdClientKeyName),
				path.Join(cfg.CertificatesDir, kubeadmconstants.EtcdCACertName))
			if err != nil {
				return false, fmt.Errorf("[etcd] Fail to retrieve client from etcd [%v]", err)
			}
			var clusterFlag = ""
			cluster := clientv3.NewCluster(client)
			ctx, cancel := context.WithTimeout(context.Background(), kubeadmconstants.TLSBootstrapTimeout)
			defer cancel()
			var memberListResponse *clientv3.MemberListResponse
			for {
				if memberListResponse, err = cluster.MemberList(ctx); err != nil {
					return false, fmt.Errorf("[etcd]  Fail to retrieve members of etcd,%s", err)
				}
				isReady := isMemberReady(memberListResponse)
				if isReady {
					break
				}
				fmt.Println("[etcd] One of etcd member is not ready")
				time.Sleep(1 * time.Second)
			}

			isCurrentMemberInCluster := false
			for _, member := range memberListResponse.Members {
				//10d549d9f5847f16: name=k8s-master-1 peerURLs=https://192.168.1.235:2380 clientURLs=http://127.0.0.1:2379,https://192.168.1.235:2379 isLeader=true
				//5039ce8be8b53ae7: name=k8s-master-2 peerURLs=https://192.168.1.236:2380 clientURLs=http://127.0.0.1:2379,https://192.168.1.236:2379 isLeader=false
				//fca36cb3ba7871c0[unstarted]: peerURLs=https://192.168.1.237:2380
				if newMemberPeerUrl != member.PeerURLs[0]  {
					clusterFlag += "," + member.Name + "=" + member.PeerURLs[0]
				} else {
					isCurrentMemberInCluster = true
				}
			}
			initialCluster += clusterFlag
			initialClusterState = "existing"
			if !isCurrentMemberInCluster {
				cluster.MemberAdd(ctx, []string{newMemberPeerUrl})
			}
			return true, nil

		})
		if err != nil {
			fmt.Printf("[etcd] Add etcd member [%s] error:[%v]\n", newMemberPeerUrl, err)
		}
	}

	defaultArguments := map[string]string{
		"name":     cfg.GetNodeName(),
		"data-dir": cfg.Etcd.Local.DataDir,

		"cert-file":             filepath.Join(cfg.CertificatesDir, kubeadmconstants.EtcdServerCertName),
		"key-file":              filepath.Join(cfg.CertificatesDir, kubeadmconstants.EtcdServerKeyName),
		"trusted-ca-file":       filepath.Join(cfg.CertificatesDir, kubeadmconstants.EtcdCACertName),
		"client-cert-auth":      "true",
		"peer-cert-file":        filepath.Join(cfg.CertificatesDir, kubeadmconstants.EtcdPeerCertName),
		"peer-key-file":         filepath.Join(cfg.CertificatesDir, kubeadmconstants.EtcdPeerKeyName),
		"peer-trusted-ca-file":  filepath.Join(cfg.CertificatesDir, kubeadmconstants.EtcdCACertName),
		"peer-client-cert-auth": "true",

		"heartbeat-interval": "500",
		"election-timeout":   "5000",
		"snapshot-count":     "10000",

		"initial-advertise-peer-urls": "https://" + advertiseAddr + ":2380",
		"listen-peer-urls":            "https://" + advertiseAddr + ":2380",
		"listen-client-urls":          "https://" + advertiseAddr + ":2379,http://" + loopback + ":2379",
		"advertise-client-urls":       "https://" + advertiseAddr + ":2379,http://" + loopback + ":2379",
		"initial-cluster-token":       "k8s",
		"initial-cluster":             initialCluster,
		"initial-cluster-state":       initialClusterState,
	}

	command := []string{"etcd"}
	command = append(command, kubeadmutil.BuildArgumentListFromMap(defaultArguments, cfg.Etcd.Local.ExtraArgs)...)
	return command
}

func isMemberReady(respone *clientv3.MemberListResponse) bool {
	for _, member := range respone.Members {
		if member.Name == "" {
			return false
		}
	}
	return true
}
