#!/usr/bin/env bash


#Deploying HA Kubernetes example
echo "Deploying Master on 192.168.1.246"
sudo bash -c "$(docker run --rm -v /tmp:/tmp 192.168.1.55/tenx_containers/tde:v3.0.0 --registry 192.168.1.55 --credential admin:vsvibpdwdssundxhhhljncnbcfieolczaeowuwggvoqkewsw   Init http://192.168.1.103:48000/api/v2/cluster/new)"

echo "Deploying HA Master on 192.168.1.247"
sudo bash -c "$(docker run --rm -v /tmp:/tmp 192.168.1.55/tenx_containers/tde:v3.0.0 --registry 192.168.1.55 --token <token> --ha-peer 192.168.1.246 Init)"

echo "Deploying HA Master on 192.168.1.248"
sudo bash -c "$(docker run --rm -v /tmp:/tmp 192.168.1.55/tenx_containers/tde:v3.0.0 --registry 192.168.1.55 --token <token> --ha-peer 192.168.1.246 Init)"


echo "Deploying LB Master"
sudo bash -c "$(docker run --rm -v /tmp:/tmp 192.168.1.55/tenx_containers/tde:v3.0.0 --registry 192.168.1.55 --token <token> --role loadbalancer Join 192.168.1.246)"

echo "Deploying LB Master"
sudo bash -c "$(docker run --rm -v /tmp:/tmp 192.168.1.55/tenx_containers/tde:v3.0.0 --registry 192.168.1.55 --token <token> --role loadbalancer Join 192.168.1.246)"

echo "Deploying Node"
sudo bash -c "$(docker run --rm -v /tmp:/tmp 192.168.1.55/tenx_containers/tde:v3.0.0 --registry 192.168.1.55 --token <token> Join 192.168.1.246)"


#Deploying Kubernetes example

echo "Deploying Master on 192.168.1.246"
sudo bash -c "$(docker run --rm -v /tmp:/tmp 192.168.1.55/tenx_containers/tde:v3.0.0 --registry 192.168.1.55 --credential admin:vsvibpdwdssundxhhhljncnbcfieolczaeowuwggvoqkewsw  Init http://192.168.1.103:48000/api/v2/cluster/new)"

echo "Deploying Node  on 192.168.1.247"
sudo bash -c "$(docker run --rm -v /tmp:/tmp 192.168.1.55/tenx_containers/tde:v3.0.0 --registry 192.168.1.55 --token <token> Join 192.168.1.246:6443)"