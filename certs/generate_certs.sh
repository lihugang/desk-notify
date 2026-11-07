#!/usr/bin/env bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
cd "$DIR"

echo "=== 1. 生成 Lihugang 个人统一根证书 (Master Root CA) ==="
cat << 'EOF' > root_ca.cnf
[ req ]
distinguished_name = req_distinguished_name
x509_extensions     = v3_ca
prompt             = no

[ req_distinguished_name ]
C  = CN
ST = Beijing
O  = Lihugang
OU = me@lihugang.top
CN = Lihugang Root CA

[ v3_ca ]
subjectKeyIdentifier   = hash
authorityKeyIdentifier = keyid:always,issuer
basicConstraints       = critical, CA:true
keyUsage               = critical, digitalSignature, cRLSign, keyCertSign
EOF

openssl genrsa -out root_ca.key 4096
openssl req -new -x509 -days 3650 -key root_ca.key -out root_ca.crt -config root_ca.cnf -sha256
openssl x509 -outform der -in root_ca.crt -out Lihugang-CA.cer

echo "=== 2. 生成传纸条 (DeskNotify) 代码签名专用证书 (由 Root CA 直接签发) ==="
cat << 'EOF' > codesign.cnf
[ req ]
distinguished_name = req_distinguished_name
prompt             = no

[ req_distinguished_name ]
C  = CN
ST = Beijing
O  = Lihugang
OU = me@lihugang.top
CN = 传纸条 (DeskNotify)

[ v3_codesign ]
basicConstraints       = CA:false
subjectKeyIdentifier   = hash
authorityKeyIdentifier = keyid,issuer
keyUsage               = critical, digitalSignature
extendedKeyUsage       = critical, codeSigning
EOF

openssl genrsa -out codesign.key 4096
openssl req -new -key codesign.key -out codesign.csr -config codesign.cnf -sha256
openssl x509 -req -days 3650 -in codesign.csr -CA root_ca.crt -CAkey root_ca.key -CAcreateserial -out codesign.crt -extfile codesign.cnf -extensions v3_codesign -sha256

echo "=== 3. 打包导出 PKCS#12 (PFX) 私钥文件 (密码: 220418ban) ==="
openssl pkcs12 -export -out codesign.pfx \
  -inkey codesign.key \
  -in codesign.crt \
  -certfile root_ca.crt \
  -name "传纸条 (DeskNotify)" \
  -passout pass:220418ban \
  -certpbe AES-256-CBC -keypbe AES-256-CBC -macalg SHA256

echo "=== 4. 生成 DeskNotify 中间 CA (用于 Go 程序首次启动动态签发 TLS 证书) ==="
cat << 'EOF' > subca.cnf
[ req ]
distinguished_name = req_distinguished_name
prompt             = no

[ req_distinguished_name ]
C  = CN
ST = Beijing
O  = Lihugang
OU = me@lihugang.top
CN = DeskNotify Intermediate CA

[ v3_subca ]
subjectKeyIdentifier   = hash
authorityKeyIdentifier = keyid:always,issuer
basicConstraints       = critical, CA:true, pathlen:0
keyUsage               = critical, digitalSignature, keyCertSign
EOF

openssl genrsa -out subca.key 2048
openssl req -new -key subca.key -out subca.csr -config subca.cnf -sha256
openssl x509 -req -days 3650 -in subca.csr -CA root_ca.crt -CAkey root_ca.key -CAcreateserial -out subca.crt -extfile subca.cnf -extensions v3_subca -sha256

echo "=== 证书链验证测试 ==="
openssl verify -CAfile root_ca.crt subca.crt
openssl verify -CAfile root_ca.crt codesign.crt

echo "=== 证书生成完毕！==="
ls -lh
