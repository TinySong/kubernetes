/*
 * Licensed Materials - Property of tenxcloud.com
 * (C) Copyright 2018 TenxCloud. All Rights Reserved.
 * 2018-10-10  @author weiwei@tenxcloud.com
 */
package controlplane

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/uuid"
	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	//tokenutil "k8s.io/kubernetes/cmd/kubeadm/app/util/token"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

func CallBack(tceAddress, tceCredential, clusterId, k8sAddress, k8sToken string) error {
	credential := strings.Split(tceCredential, ":")
	if len(credential) < 2 {
		return fmt.Errorf("[callback] There's not valide cridiential for TenxCloud Enterprise Server, please provide correct one \n")
	}
	params := make(map[string]string)
	id := randInt(0, 1000)
	const CLUSTER_NAME_PREFIX = "k8s-"
	params["name"] = CLUSTER_NAME_PREFIX + strconv.Itoa(id)
	if "" != clusterId {
		params["cluster_id"] = clusterId
	}
	params["access_url"] = fmt.Sprintf("%s:6443", k8sAddress)
	params["token"] = k8sToken
	params["description"] = CLUSTER_NAME_PREFIX + strconv.Itoa(id)
	playload, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("[callback] Failed to consturct an HTTP request [%v] \n", err)
	}
	req, err := http.NewRequest("POST", tceAddress, bytes.NewBuffer(playload))
	if err != nil {
		return fmt.Errorf("[callback] Failed to consturct an HTTP request [%v] \n", err)
	}
	req.Header.Set("username", credential[0])
	req.Header.Set("authorization", fmt.Sprintf("token %s", credential[1]))
	//if the url is https, will skip the verify
	client := &http.Client{}
	if uri, err := url.Parse(tceAddress); err == nil && uri.Scheme == "https" {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{Transport: tr}
	}
	response, err := client.Do(req)
	defer response.Body.Close()
	if err != nil {
		return fmt.Errorf("[callback] Failed to callback Kubernetes Enterprise Platform to register the cluster [%v] \n ", err)
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("[callback] Failed to ReadAll cluster info [%v],Body: %v \n", err, string(body))
	}
	fmt.Println("[callback] This Kubernetes cluster registered to Kubernetes Enterprise Platform Successfully \n ")
	return nil
}

func randInt(min int, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	return min + rand.Intn(max-min)
}

// Create long-term token for TenxCloud Enterprise Server in /etc/kubernetes/pki/tokens.csv
func CreateTokenAuthFile(cfg  *kubeadmapi.InitConfiguration) error {
	bts, _:= kubeadmapi.NewBootstrapTokenString(cfg.BootstrapTokens[0].Token.String())
	tokenAuthFilePath := path.Join(cfg.CertificatesDir, "tokens.csv")
	serialized := []byte(fmt.Sprintf("%s,admin,%s,kubernetes\n", bts.Secret, uuid.NewUUID()))
	// DumpReaderToFile create a file with mode 0600
	if err := cmdutil.DumpReaderToFile(bytes.NewReader(serialized), tokenAuthFilePath); err != nil {
		return fmt.Errorf("Failed to save token auth file (%q) [%v]\n", tokenAuthFilePath, err)
	}
	return nil
}
