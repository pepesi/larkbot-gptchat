# 部署在k3s上

### 提前要做的事情

0. 克隆这个代码库

1. 准备的应用，机器人，准备好 appid, appsecret 等

2. openai 大陆是不可用的, 所以（海外机器 or 梯子)得有一个

3. 修改 config.yaml ，将里头的变量换成你自己的

### 部署

1. 安装k3s

参考: https://docs.k3s.io/installation

2. 构建镜像 

REV=latest ./images.sh

3. 部署到k3s中

export MYNAMESPACE=default


kubectl create ns ${MYNAMESPACE}


helm template yamls | kubectl -n ${MYNAMESPACE} apply -f -


4. 访问firefox， 安装agent插件，tabrealoader插件

agent的配置请参考作者的文档 `https://github.com/acheong08/ChatGPT-API-agent`

tabreloader  插件是为了在挂的时候，自动刷新
