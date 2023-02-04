# 部署在k3s上

### 提前要做的事情

0. 克隆这个代码库

1. 准备的应用，机器人，准备好 appid, appsecret, openai的api_key (https://platform.openai.com/account/api-keys) 等

2. openai 大陆是不可用的, 所以（海外机器 or 梯子)得有一个

3. 修改 config.yaml ，将里头的变量换成你自己的


### 部署

docker-compose up -d