#!/bin/bash

VER=${VER:-latest}

# build lark bot server image
docker build -t larkbot:${VER} .

# if use k3s to deploy, import images into k3s nodes
if [[ ${K3S} ]];then
    # import larkbot image
    docker save --output larkbot-${VER}.tar larkbot:${VER}
    k3s ctr images import larkbot-${VER}.tar
fi
