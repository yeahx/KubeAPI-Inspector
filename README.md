# KubeAPI-Inspector:discover the secrets hidden in apis
English | [简体中文](https://github.com/yeahx/KubeAPI-Inspector/blob/main/README_zh.md)
## Description

A tool specifically designed for Kubernetes environments aims to efficiently and automatically discover hidden vulnerable APIs within clusters. It reveals and demonstrates a common error through a workshop format, which could lead to API endpoint authentication failures and potentially compromise the entire cluster. The workshop can be deployed using Kubernetes resource YAML files.

![demo](https://github.com/yeahx/KubeAPI-Inspector/blob/main/demo.gif)
## Features
### Implemented in Inspector
* 【✅】Automatically parse OpenAPI to identify sensitive fields
* 【✅】Automatically detect potential authentication bypass APIs
* 【✅】Automatically load credentials from the environment
### To be Implemented in Inspector
* 【 】Automatically discover services and detect potential vulnerabilities in extension API servers
* 【 】exploitation of known control plane components?
### Implemented in Workshop
* 【✅】Flawed implementation of the REST layer
### To be Implemented in Workshop
* 【 】Typical vulnerabilities involving operator controllers

## Usage
### in-cluster
1. download binary in pod
2. run binary `./inspector`
### out-of-cluster
1. `./inspector -kubeconfig path/to/kubeconfig`
2. test other namespace `./inspector -kubeconfig path/to/kubeconfig -namespace kube-system`
3. skip sensitive field test `./inspector -kubeconfig path/to/kubeconfig -skipCheckSensitiveField=true`

## Installation
### Requirements
1. golang>1.22
2. kubernetes and docker
3. linux-amd64, linux-arm
### build kubeapi-inspector
CWD: /repo/
1. use go build `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -o inspector cmd/inspector/main.go`
2. or use docker to build `docker build . -t inspector:latest`
### build & deploy workshop steps
CWD: /repo/workshop/
1. setup a kubernetes cluster, maybe you should use minikube, e.g. `minikube start --kubernetes-version='v1.23.17'`
2. build workshop image with docker `docker build . -t workshop-apiserver:latest`
3. deploy etcd for workshop-apiserver `cd workshop/examples/etcd && ./generate-certs.sh && deploy.sh`
4. create workshop k8s resource `cat examples/{namespace,apiserviceservice,workshop-apiserver-sa,workshop-apiserver-clusterrolebinding,workshop-apiserver-deployment}.yaml | kubectl apply -f -`
5. create demo cluster resource and tenant accounts `kubectl apply -f examples/tenant`

## License
MIT License