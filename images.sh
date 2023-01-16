#!/bin/bash

VER=${VER:-latest}

# build chat-server image
docker build -t chat-server:${VER} Dockerfiles/server

# build lark bot server image
docker build -t larkbot:${VER} .

# if use k3s to deploy, import images into k3s nodes
if [[ ${K3S} ]];then
    # import chat-server image
    docker save --output chat-server-${VER}.tar chat-server:${VER}
    k3s ctr images import chat-server-${VER}.tar
    # import larkbot image
    docker save --output larkbot-${VER}.tar larkbot:${VER}
    k3s ctr images import larkbot-${VER}.tar
fi
