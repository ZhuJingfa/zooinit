#构建命令：docker build --no-cache -t haimi:zooinit-bootstrap .
#run:
#version: 0.1.0
#version: 0.1.0 zooinit+etcd package
# Copyright 2016 祝景法(Bruce)@haimi.com. www.haimi.com All rights reserved.

#need python, can't scratch
FROM haimi:base

#MAINTAINER bruce@haimi.com

WORKDIR /server/zooinit/

RUN axel http://docker.alishui.com/etcd-v2.2.2-linux-amd64.tar.gz && tar xzvf etcd-v2.2.2-linux-amd64.tar.gz \
    && mv etcd-v2.2.2-linux-amd64/etcd* /usr/local/bin && rm -rf etcd-v2.2.2*

ENV PATH=$PATH:/server/zooinit/bin

ADD transfer/ /server/zooinit/

VOLUME ["/data"]

#etcd client port
EXPOSE 2379
ENTRYPOINT ["/server/zooinit/entrypoint.sh"]