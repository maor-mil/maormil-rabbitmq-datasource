#!/bin/sh
# Generates THROWAWAY TLS material for the local/E2E RabbitMQ broker (run by the `rabbitmq-certs`
# docker compose service). Never use these certificates anywhere else.
#
#   ca.pem / ca-key.pem             test CA that signs the server and client certificates
#   other-ca.pem                    an unrelated CA, to test certificate verification failures
#   server.pem / server-key.pem     broker certificate (SAN: rabbitmq, localhost, 127.0.0.1)
#   client.pem / client-key.pem     client certificate; its CN (e2e-tls-client) is the RabbitMQ user
#                                   for SASL EXTERNAL authentication
set -eu
OUT="${1:-/certs}"
CLIENT_CN="${CLIENT_CN:-e2e-tls-client}"
DAYS=3650

mkdir -p "$OUT"
cd "$OUT"
if [ -f .complete ]; then
  echo "TLS test certificates already exist in $OUT"
  exit 0
fi

openssl req -x509 -newkey rsa:2048 -nodes -days "$DAYS" -keyout ca-key.pem -out ca.pem \
  -subj "/CN=maormil-rabbitmq-datasource test CA" \
  -addext "basicConstraints=critical,CA:TRUE" -addext "keyUsage=critical,keyCertSign,cRLSign"
openssl req -x509 -newkey rsa:2048 -nodes -days "$DAYS" -keyout other-ca-key.pem -out other-ca.pem \
  -subj "/CN=unrelated test CA" \
  -addext "basicConstraints=critical,CA:TRUE" -addext "keyUsage=critical,keyCertSign,cRLSign"

printf 'subjectAltName=DNS:rabbitmq,DNS:localhost,IP:127.0.0.1\nextendedKeyUsage=serverAuth\n' > server.ext
openssl req -newkey rsa:2048 -nodes -keyout server-key.pem -out server.csr -subj "/CN=rabbitmq"
openssl x509 -req -in server.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial -days "$DAYS" -out server.pem -extfile server.ext

printf 'extendedKeyUsage=clientAuth\n' > client.ext
openssl req -newkey rsa:2048 -nodes -keyout client-key.pem -out client.csr -subj "/CN=$CLIENT_CN"
openssl x509 -req -in client.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial -days "$DAYS" -out client.pem -extfile client.ext

rm -f ./*.csr ./*.ext ./*.srl
# Test-only material: must be readable by the broker (uid 999) and by the test runner on the host.
chmod 644 ./*.pem
touch .complete
echo "Generated TLS test certificates in $OUT"
