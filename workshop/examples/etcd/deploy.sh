#!/bin/bash
set -e

# 检查证书是否已经生成
if [ ! -d "./certs" ]; then
  echo "error: certs directory not found"
  echo "plz run ./generate-certs.sh to generate certificates first"
  exit 1
fi

# 确保命名空间存在
kubectl create namespace kubeapi-inspector-workshop --dry-run=client -o yaml | kubectl apply -f -

# 创建服务账号
kubectl create serviceaccount workshop-apiserver-sa -n kubeapi-inspector-workshop --dry-run=client -o yaml | kubectl apply -f -

# 创建证书的Secret
kubectl create secret generic etcd-certs \
  --from-file=ca.crt=certs/ca.crt \
  --from-file=server.crt=certs/server.pem \
  --from-file=server.key=certs/server-key.pem \
  --from-file=etcd-client.crt=certs/etcd-client.pem \
  --from-file=etcd-client.key=certs/etcd-client-key.pem \
  -n kubeapi-inspector-workshop \
  --dry-run=client -o yaml | kubectl apply -f -

# 创建etcd服务器地址的ConfigMap
kubectl create configmap etcd-config \
  --from-literal=etcd-servers=https://etcd.kubeapi-inspector-workshop.svc.cluster.local:2379 \
  -n kubeapi-inspector-workshop \
  --dry-run=client -o yaml | kubectl apply -f -

# 部署etcd
kubectl apply -f etcd-deployment.yaml
kubectl apply -f etcd-service.yaml

# 部署workshop-apiserver
kubectl apply -f ../workshop-apiserver-deployment.yaml

echo "deployment completed!"
echo "you can use the following command to check the deployment status:"
echo "kubectl get pods,svc -n kubeapi-inspector-workshop" 