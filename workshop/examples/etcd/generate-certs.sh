#!/bin/bash
set -e

#检查cfssl是否已安装
if ! command -v cfssl &> /dev/null || ! command -v cfssljson &> /dev/null; then
    echo "plz install cfssl and cfssljson first"
    echo "Linux x86_64 example:"
    echo "curl -L https://github.com/cloudflare/cfssl/releases/download/v1.6.1/cfssl_1.6.1_linux_amd64 -o cfssl"
    echo "curl -L https://github.com/cloudflare/cfssl/releases/download/v1.6.1/cfssljson_1.6.1_linux_amd64 -o cfssljson"
    echo "chmod +x cfssl cfssljson"
    echo "sudo mv cfssl cfssljson /usr/local/bin/"
    exit 1
fi

# 创建证书输出目录
CERT_DIR="./certs"
mkdir -p ${CERT_DIR}
cd ${CERT_DIR}

# 生成CA证书
echo "generating CA certificate..."
echo '{"CN":"CA","key":{"algo":"rsa","size":2048}}' | cfssl gencert -initca - | cfssljson -bare ca
mv ca.pem ca.crt
mv ca-key.pem ca.key

# 创建证书配置文件
cat > config.json <<EOF
{
  "signing": {
    "default": {
      "expiry": "8760h"
    },
    "profiles": {
      "server": {
        "usages": ["signing", "key encipherment", "server auth"],
        "expiry": "8760h"
      },
      "client": {
        "usages": ["signing", "key encipherment", "client auth"],
        "expiry": "8760h"
      }
    }
  }
}
EOF

# 生成etcd服务端证书
echo "generating etcd server certificate..."
cat > server-csr.json <<EOF
{
  "CN": "etcd",
  "hosts": [
    "etcd.kubeapi-inspector-workshop.svc.cluster.local",
    "etcd",
    "127.0.0.1"
  ],
  "key": {
    "algo": "rsa",
    "size": 2048
  }
}
EOF
cfssl gencert -ca=ca.crt -ca-key=ca.key -config=config.json -profile=server server-csr.json | cfssljson -bare server

# 生成etcd客户端证书
echo "generating etcd client certificate..."
cat > client-csr.json <<EOF
{
  "CN": "client",
  "key": {
    "algo": "rsa",
    "size": 2048
  }
}
EOF
cfssl gencert -ca=ca.crt -ca-key=ca.key -config=config.json -profile=client client-csr.json | cfssljson -bare etcd-client

# 显示生成的证书
echo "certificate generation completed!"
echo "generated certificate files:"
ls -la

# 提供kubectl命令以创建Secret和ConfigMap
echo ""
echo "create kubernetes secret and configmap commands:"
echo "kubectl create namespace kubeapi-inspector-workshop # if namespace not exists"
echo "kubectl create secret generic etcd-certs \\"
echo "  --from-file=ca.crt=certs/ca.crt \\"
echo "  --from-file=server.crt=certs/server.pem \\"
echo "  --from-file=server.key=certs/server-key.pem \\"
echo "  --from-file=etcd-client.crt=certs/etcd-client.pem \\"
echo "  --from-file=etcd-client.key=certs/etcd-client-key.pem \\"
echo "  -n kubeapi-inspector-workshop"
echo ""
echo "kubectl create configmap etcd-config \\"
echo "  --from-literal=etcd-servers=https://etcd.kubeapi-inspector-workshop.svc.cluster.local:2379 \\"
echo "  -n kubeapi-inspector-workshop" 