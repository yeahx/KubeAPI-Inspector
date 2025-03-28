# KubeAPI-Inspector
[English](https://github.com/yeahx/KubeAPI-Inspector/blob/main/README.md) | 简体中文
## 概述

一个专为 Kubernetes 环境设计的工具，旨在高效且自动地发现集群中隐藏的漏洞 API。 
附赠一个靶场可以学习到自定义 apiserver 的一个经典漏洞，这一设计会导致 API 端点鉴权失效，并可能危及整个集群。

![demo](https://github.com/yeahx/KubeAPI-Inspector/blob/main/demo.gif)
## 功能
### inspector 已实现
* 【✅】自动解析 openapi 发现敏感字段
* 【✅】自动探测潜在的认证绕过的 api
* 【✅】自动加载环境内的凭证
### inspector 待实现
* 【 】自动服务发现并探测潜在的缺陷 extension apiserver
* 【 】集成已知控制面组件利用？
### workshop 已实现
* 【✅】REST层的缺陷实现
### workshop 待实现
* 【 】带有 operator 控制器的典型漏洞
## 使用方法
### 集群内
1. 在 pod 中下载二进制文件
2. 运行二进制文件 `./inspector`
### 集群外
1. ./inspector -kubeconfig path/to/kubeconfig
2. 测试其他命名空间 ./inspector -kubeconfig path/to/kubeconfig -namespace kube-system
3. 跳过敏感字段测试 ./inspector -kubeconfig path/to/kubeconfig -skipCheckSensitiveField=true
## 安装
### 要求
1. golang>1.22
2. kubernetes 和 docker
3. linux-amd64, linux-arm
### 构建 kubeapi-inspector
* 当前工作目录：/repo/
1. 使用 `go build CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -o inspector cmd/inspector/main.go`
2. 或使用 docker 构建 `docker build . -t inspector:latest`
### 构建和部署 workshop 步骤
* 当前工作目录：/repo/workshop/

1. 设置一个 Kubernetes 集群，可以使用 minikube，如 `minikube start --kubernetes-version='v1.23.17'`
2. 使用 docker 构建 workshop 镜像 `docker build . -t workshop-apiserver:latest`
3. 部署 workshop-apiserver 使用的etcd `cd workshop/examples/etcd && ./generate-certs.sh && deploy.sh`
4. 创建 workshop k8s 资源 `cat examples/{namespace,apiserviceservice,workshop-apiserver-sa,workshop-apiserver-clusterrolebinding,workshop-apiserver-deployment}.yaml | kubectl apply -f -`
5. 创建 demo 的集群 resource 及租户的服务账号 `kubectl apply -f examples/tenant`