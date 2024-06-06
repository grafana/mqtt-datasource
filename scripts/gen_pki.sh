#/usr/bin/env bash

openssl genrsa 2048 > testdata/ca-key.pem

openssl req -new -x509 -nodes -days 365000 \
   -subj "/CN=localhost/O=MQTT" \
   -key testdata/ca-key.pem \
   -out testdata/ca-cert.pem

openssl req -x509 -newkey rsa:4096 -sha256 -days 365000 \
   -set_serial 01 \
   -nodes -keyout testdata/server-key.pem \
   -out testdata/server-cert.pem \
   -subj "/CN=localhost/O=MQTT" \
   -addext "subjectAltName=DNS:localhost,IP:127.0.0.1" \
   -addext "extendedKeyUsage = serverAuth"\
   -CA testdata/ca-cert.pem \
   -CAkey testdata/ca-key.pem 

 openssl req -x509 -newkey rsa:4096 -sha256 -days 365000 \
   -set_serial 02 \
   -nodes -keyout testdata/client-key.pem \
   -out testdata/client-cert.pem \
   -subj "/CN=client/O=MQTT" \
   -addext "extendedKeyUsage = clientAuth" \
   -CA testdata/ca-cert.pem \
   -CAkey testdata/ca-key.pem