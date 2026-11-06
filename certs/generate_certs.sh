#!/usr/bin/env bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
cd "$DIR"

echo "=== 1. 生成 DeskNotify 统一根证书 (Root CA) ==="
cat << 'EOF' > ca.cnf
[ req ]
distinguished_name = req_distinguished_name
x509_extensions     = v3_ca
prompt             = no

[ req_distinguished_name ]
C  = CN
ST = Beijing
O  = DeskNotify (传纸条)
OU = me@lihugang.top
CN = DeskNotify Root CA

[ v3_ca ]
subjectKeyIdentifier   = hash
authorityKeyIdentifier = keyid:always,issuer
basicConstraints       = critical, CA:true
keyUsage               = critical, digitalSignature, cRLSign, keyCertSign
EOF

openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 3650 -key ca.key -out ca.crt -config ca.cnf -sha256
openssl x509 -outform der -in ca.crt -out DeskNotify-CA.cer

echo "=== 2. 生成代码签名专用证书 (Authenticode Code Signing) ==="
cat << 'EOF' > codesign.cnf
[ req ]
distinguished_name = req_distinguished_name
prompt             = no

[ req_distinguished_name ]
C  = CN
ST = Beijing
O  = DeskNotify
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
openssl x509 -req -days 3650 -in codesign.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out codesign.crt -extfile codesign.cnf -extensions v3_codesign -sha256

echo "=== 3. 打包导出 PKCS#12 (PFX) 私钥文件 (密码: 220418ban) ==="
openssl pkcs12 -export -out codesign.pfx \
  -inkey codesign.key \
  -in codesign.crt \
  -certfile ca.crt \
  -name "传纸条 (DeskNotify)" \
  -passout pass:220418ban \
  -certpbe AES-256-CBC -keypbe AES-256-CBC -macalg SHA256

echo "=== 证书生成完毕！==="
ls -lh
