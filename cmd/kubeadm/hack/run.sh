#!/bin/bash
REGISTRY_SERVER="index.tenxcloud.com"
REGISTRY_USER="system_containers"
AGENT_VERSION="v4.0.0"
K8S_VERSION="v1.12.3"
ETCD_VERSION="3.2.24"
CALICO_VERSION="v3.4.0"
ROLE="node"
welcome() {
message="$(cat <<EOF
*******************************************************
||                                                   ||
||                                                   ||
||        Kubernetes Enterprise Edition              ||
||                                                   ||
||                                                   ||
***********************************Kubernetes Enterprise
Welcome to Kubernetes Enterprise Edition Deployment Engine
EOF
)"
echo "welcome() {"
echo "  cat <<\"EOF\""
echo "$message"
echo "EOF"
echo "}"
}

Usage() {
cat <<EOF
Kubernetes Enterprise Edition Deployment Engine\n
\n
Command: \n
    [Option] Join <Master> \n
    [Option] Init [TCEAddress] \n
    [Option] Uninstall \n
\n
Option:\n
    --registry       \t Registry server, default is index.tenxcloud.com \n
    --address        \t Advertised address of the current machine, if not set, it will get a one automatically\n
    --version        \t Cluster version that will be deployed\n
    --token          \t kubernetes token \n
    --credential     \t credential to access tce api server\n
    --ha-peer        \t Peer master in HA mode\n
    --role           \t Role of current machine: master, node, loadbalancer
EOF
}



Clean=$(cat <<EOF
  Clean() {
    cp /kubeadm  /tmp/  1>/dev/null 2>&1
    #remove agent
    echo "Cleaning previous agent if existing"
    docker stop agent   1>/dev/null 2>&1
    docker rm -f agent  1>/dev/null 2>&1
    /tmp/kubeadm reset -f
  }
EOF
)
eval "${Clean}"


PullImage=$(cat <<EOF
  PullImage() {
  echo "Pulling Necessary Images from \${1}"
  if [ \${3} == "master" ]; then
      docker pull \${1}/\${2}/hyperkube-amd64:${K8S_VERSION}
      docker pull \${1}/\${2}/agent:${AGENT_VERSION}
      docker pull  \${1}/\${2}/kubectl-amd64:${K8S_VERSION}
      docker pull  \${1}/\${2}/etcd:${ETCD_VERSION}
      docker pull  \${1}/\${2}/ctl:${CALICO_VERSION}
  else
      docker pull \${1}/\${2}/hyperkube-amd64:${K8S_VERSION}
      docker pull \${1}/\${2}/agent:${AGENT_VERSION}
      docker pull \${1}/\${2}/kubectl-amd64:${K8S_VERSION}
      docker pull \${1}/\${2}/ctl:${CALICO_VERSION}
  fi
  }
EOF
)


uninstall(){
#copy kubeadm from containers to /tmp
cp /kubeadm  /tmp/  > /dev/null 2>&1

cat <<EOF
#!/bin/bash
${Clean}
Clean
rm /tmp/kubeadm 2>/dev/null
echo "Uninstall Node Successfully"
EOF
}



#Deploy kubernetes master
Master() {
  #copy kubeadm from containers to /tmp
  cat > /tmp/init.yaml << EOF
apiVersion: kubeadm.k8s.io/v1alpha3
kind: ClusterConfiguration
kubernetesVersion: ${K8S_VERSION}
imageRepository: ${REGISTRY_SERVER}/${REGISTRY_USER}
unifiedControlPlaneImage: ${REGISTRY_SERVER}/${REGISTRY_USER}/hyperkube-amd64:${K8S_VERSION}
EOF
  cp /kubeadm  /tmp/  1>/dev/null 2>&1
  ADVERTISE_ADDRESSES_AGENT=""
  if [ -n "${ADDRESS}" ]; then
    ADVERTISE_ADDRESSES_AGENT="--advertise-address=${ADDRESS}"
  fi


  #master ha peer begin
  if [ -n "${HA_PEER}" ]; then
    if [ -z "${K8S_TOKEN}" ]; then
       cat <<EOF
#!/bin/bash
echo "For HA mode, Please set kubernetes token with parameter --token <tokenstring>"
EOF
       exit 1
    fi

    cat >> /tmp/init.yaml << EOF
bootstrapTokens:
- token: ${K8S_TOKEN}
EOF

    # add TokenAPIServers
    cat >> /tmp/init.yaml << EOF
discoveryTokenAPIServers:
EOF
    local apiserver=${HA_PEER//,/ }
    local file=$(mktemp /tmp/servers.XXXXXXXX)
    for addr in $apiserver; do
       echo "- $addr:6443" >> $file
    done
    cat $file >> /tmp/init.yaml
    rm -rf $file


    cat <<EOF
#!/bin/bash
$(welcome)
welcome
${Clean}
Clean

${PullImage}
PullImage ${REGISTRY_SERVER} ${REGISTRY_USER}  "master"

result=0
/tmp/kubeadm init --config /tmp/init.yaml
result=\$?
rm -rf $(which kubeadm)
mv /tmp/kubeadm /usr/bin/  >/dev/null

docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/kubectl-amd64:${K8S_VERSION} /bin/kubectl /tmp
rm -rf $(which kubectl)
mv /tmp/kubectl /usr/bin/  >/dev/null

docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/etcd:${ETCD_VERSION}  /usr/local/bin/etcdctl /tmp
rm -rf $(which etcdctl)
mv /tmp/etcdctl /usr/bin/  >/dev/null

docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/ctl:${CALICO_VERSION} /calicoctl /tmp
rm -rf $(which calicoctl)
mv /tmp/calicoctl /usr/bin/  >/dev/null

docker run --net=host -d --cpu-period=100000 --cpu-quota=100000 --memory=100000000 --restart=always  -v /tmp:/tmp  -v /etc/hosts:/etc/hosts -v /etc/kubernetes:/etc/kubernetes  -v /etc/resolv.conf:/etc/resolv.conf   --name agent  ${REGISTRY_SERVER}/${REGISTRY_USER}/agent:${AGENT_VERSION}  --role=master --etcd-servers=http://127.0.0.1:2379 ${ADVERTISE_ADDRESSES_AGENT} --dns-enable=true --ssl-enable=false >/dev/null
result=\$?
if [ \${result} -eq 0  ];then
   echo "Kubernetes Enterprise Edition cluster deployed successfully"
else
   echo "Kubernetes Enterprise Edition cluster deployed  failed!"
fi

EOF

    exit 0
  #master ha peer end
  fi

  #Normal master mode
  if [ -n "${SERVER_URL}" ] && [ -n "${CREDENTIAL}" ]; then
    if [ -n "${CLUSTERID}" ]; then
       cat >> /tmp/init.yaml << EOF
apiServerUrl: ${SERVER_URL}
apiServerCredential: ${CREDENTIAL}
clusterId: ${CLUSTERID}
EOF
    else
       cat >> /tmp/init.yaml << EOF
apiServerUrl: ${SERVER_URL}
apiServerCredential: ${CREDENTIAL}
EOF
    fi
  fi

  if [  -n "${NETWORK}" -o  -n "${POD_CIDR}" -o -n "${SERVICE_CIDR}" -o -n "${SERVICE_DNS_DOMAIN}" ]; then
     cat >> /tmp/init.yaml << EOF
networking:
EOF
  fi

  if [ -n "${NETWORK}" ]; then
     cat >> /tmp/init.yaml << EOF
  plugin: ${NETWORK}
EOF
  fi
  if [ -n "${POD_CIDR}" ]; then
    cat >> /tmp/init.yaml << EOF
  podSubnet: ${POD_CIDR}
EOF
  fi

  if [ -n "${SERVICE_CIDR}" ]; then
    cat >> /tmp/init.yaml << EOF
  serviceSubnet: ${SERVICE_CIDR}
EOF
  fi

  if [ -n "${SERVICE_DNS_DOMAIN}" ]; then
    cat >> /tmp/init.yaml << EOF
  dnsDomain: ${SERVICE_DNS_DOMAIN}
EOF
  fi

  cat <<EOF
#!/bin/bash
$(welcome)
welcome
${Clean}
Clean

${PullImage}
PullImage ${REGISTRY_SERVER} ${REGISTRY_USER}  "master"

result=0
/tmp/kubeadm init  --config /tmp/init.yaml
result=\$?
rm -rf $(which kubeadm)
mv /tmp/kubeadm /usr/bin/  >/dev/null

docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/kubectl-amd64:${K8S_VERSION} /bin/kubectl /tmp
rm -rf $(which kubectl)
mv /tmp/kubectl /usr/bin/  >/dev/null

docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/etcd:${ETCD_VERSION}  /usr/local/bin/etcdctl /tmp
rm -rf $(which etcdctl)
mv /tmp/etcdctl /usr/bin/  >/dev/null

docker run --rm -v /tmp:/tmp --entrypoint cp  ${REGISTRY_SERVER}/${REGISTRY_USER}/ctl:${CALICO_VERSION} /calicoctl /tmp
rm -rf $(which calicoctl)
mv /tmp/calicoctl /usr/bin/  >/dev/null

docker run --net=host -d --cpu-period=100000 --cpu-quota=100000 --memory=100000000 --restart=always -v /tmp:/tmp -v /etc/kubernetes:/etc/kubernetes -v /etc/hosts:/etc/hosts -v /etc/resolv.conf:/etc/resolv.conf --name agent  ${REGISTRY_SERVER}/${REGISTRY_USER}/agent:${AGENT_VERSION} ${ADVERTISE_ADDRESSES_AGENT} --role=master --etcd-servers=http://127.0.0.1:2379 --dns-enable=true  --ssl-enable=false >/dev/null
result=\$?
if [ \${result} -eq 0  ];then
   echo "Kubernetes Enterprise Edition cluster deployed successfully"
else
   echo "Kubernetes Enterprise Edition cluster deployed  failed!"
fi
EOF
exit 0
#Normal master mode end
}



Node() {
  #copy kubeadm from containers to /tmp
  cp /kubeadm  /tmp/ 2>/dev/null

  ADVERTISE_ADDRESSES_AGENT=""
  if [ -n "${ADDRESS}" ]; then
    ADVERTISE_ADDRESSES_AGENT="--advertise-address=${ADDRESS}"
  fi

  if [ -z "${K8S_TOKEN}" ]; then
    cat <<EOF
#!/bin/bash
echo "Please set kubernetes token with parameter --token <tokenstring>"
EOF
    exit 1
  fi

  ## loadbalancer node
  if [ "$ROLE" = "loadbalancer" ]; then
    cat <<EOF
#!/bin/bash
$(welcome)
welcome
${Clean}
Clean

result=0
echo "Deploying loadbalancer..."
docker run --net=host -d --cpu-period=100000 --cpu-quota=100000 --memory=100000000 --restart=always -v /tmp:/tmp  -v /etc/hosts:/etc/hosts -v /etc/kubernetes:/etc/kubernetes  -v /etc/resolv.conf:/etc/resolv.conf --name agent ${REGISTRY_SERVER}/${REGISTRY_USER}/agent:${AGENT_VERSION}  ${ADVERTISE_ADDRESSES_AGENT} --role=loadbalancer --etcd-servers=https://${MASTER}:2379 --accesstoken=${K8S_TOKEN} --cert-servers=${MASTER} --cert-dir=/etc/kubernetes/pki --dns-enable=false --ssl-enable=true >/dev/null
result=\$?
if [ \${result} -eq 0  ];then
   echo "Kubernetes Enterprise Edition cluster deployed successfully"
else
   echo "Kubernetes Enterprise Edition cluster deployed  failed!"
fi
EOF

   return
  fi



    cat > /tmp/join.yaml << EOF
apiVersion: kubeadm.k8s.io/v1alpha3
kind: JoinConfiguration
token: ${K8S_TOKEN}
EOF
  ## Normal slave node
  if [ -z "${CA_CERT_HASH}" ]; then
      cat <<EOF
#!/bin/bash
echo "Please set kubernetes root ca cert hash with parameter --ca-cert-hash sha256:<hash>"
EOF
      exit 1
  fi

    cat >> /tmp/join.yaml << EOF
discoveryTokenCACertHashes:
- ${CA_CERT_HASH}
EOF

    # add TokenAPIServers
    cat >> /tmp/join.yaml << EOF
discoveryTokenAPIServers:
EOF
    local apiserver=${MASTER//,/ }
    local file=$(mktemp /tmp/servers.XXXXXXXX)
    for addr in $apiserver; do
       echo "- $addr:6443" >> $file
    done
    cat $file >> /tmp/join.yaml
    rm -rf $file

  cat <<EOF
#!/bin/bash
$(welcome)
welcome
${Clean}
Clean

${PullImage}
PullImage ${REGISTRY_SERVER} ${REGISTRY_USER}  "node"
result=0
/tmp/kubeadm join --config /tmp/join.yaml
result=\$?
rm -rf /tmp/kubeadm > /dev/null 2>&1
docker run --net=host -d --cpu-period=100000 --cpu-quota=100000 --memory=100000000 --restart=always  -v /tmp:/tmp -v /etc/hosts:/etc/hosts -v /etc/kubernetes:/etc/kubernetes  -v /etc/resolv.conf:/etc/resolv.conf --name agent  ${REGISTRY_SERVER}/${REGISTRY_USER}/agent:${AGENT_VERSION} ${ADVERTISE_ADDRESSES_AGENT} --role=node --etcd-servers=https://${MASTER}:2379 --dns-enable=true --cert-dir=/etc/kubernetes/pki  --accesstoken=${K8S_TOKEN} --cert-servers=${MASTER}  --ssl-enable=true > /dev/null
result=\$?
if [ \${result} -eq 0  ];then
   echo "Kubernetes Enterprise Edition cluster deployed successfully"
else
   echo "Kubernetes Enterprise Edition cluster deployed  failed!"
fi
EOF
exit 0
}



# if there's no valid parameter, it will show help message
if [ "$#" -le 0 ] ; then
  echo -e $(Usage)
  exit 0
fi
#welcome message

#dispatch different parameters
 while(( $# > 0 ))
    do
        case "$1" in
          "--registry" )
              REGISTRY_SERVER="$2"
              shift 2;;
          "--address" )
              ADDRESS="$2"
              shift 2;;
          "--version" )
              K8S_VERSION="$2"
              shift 2 ;;
          "--token" )
              K8S_TOKEN="$2"
              shift 2 ;;
          "--ca-cert-hash" )
              CA_CERT_HASH="$2"
              shift 2 ;;
          "--credential" )
              CREDENTIAL="$2"
              shift 2 ;;
          "--clusterId" )
              CLUSTERID="$2"
              shift 2 ;;
          "--ha-peer" )
              HA_PEER="$2"
              shift 2 ;;
          "--role" )
              ROLE="$2"
              shift 2 ;;
          "Join" )
              if [ "$#" -le 1 ]; then
                echo "Please Enter Master Address and Auth Token"
                exit
              fi
              MASTER="$2"
              Node
              exit 0
              shift 3;;
          "Init" )
              if [ "$#" -gt 1 ]; then
                SERVER_URL="$2"
              fi
              Master
              exit 0
              shift 2;;
          "Uninstall" )
              uninstall
              exit 0
              shift 1;;
          "welcome" )
              exit 0
              shift 1;;
            * )
                #echo "Invalid parameter: $1"
                echo -e $(Usage)
                exit 1
        esac
    done # end while