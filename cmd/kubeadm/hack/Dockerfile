FROM index.tenxcloud.com/google_containers/alpine-with-bash:3.7

MAINTAINER  "Joshua Andrew"   weiwei@tenxcloud.com

ADD ./kubeadm /kubeadm

COPY run.sh /
RUN chmod 755 run.sh
VOLUME ["/tmp","/tmp"]

ENTRYPOINT ["/bin/bash", "/run.sh"]