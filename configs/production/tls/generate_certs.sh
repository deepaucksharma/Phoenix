#!/bin/bash

# Generate TLS certificates for production Phoenix deployment
# This script creates a CA and server certificates for mTLS

set -euo pipefail

CERT_DIR="/home/deepak/phoenix-vnext/configs/production/tls"
VALIDITY_DAYS=365

echo "Generating Phoenix Production TLS Certificates..."

# Create CA private key
openssl genrsa -out "$CERT_DIR/ca-key.pem" 4096

# Create CA certificate
openssl req -new -x509 -days $VALIDITY_DAYS -key "$CERT_DIR/ca-key.pem" \
  -out "$CERT_DIR/ca.crt" \
  -subj "/C=US/ST=CA/L=San Francisco/O=Phoenix/OU=Platform/CN=Phoenix CA"

# Create server private key
openssl genrsa -out "$CERT_DIR/server-key.pem" 4096

# Create server certificate request
openssl req -new -key "$CERT_DIR/server-key.pem" \
  -out "$CERT_DIR/server.csr" \
  -subj "/C=US/ST=CA/L=San Francisco/O=Phoenix/OU=Platform/CN=phoenix-collector"

# Create extensions file for SAN
cat > "$CERT_DIR/server-extfile.cnf" <<EOF
subjectAltName = DNS:phoenix-collector,DNS:*.phoenix-collector,DNS:localhost,IP:127.0.0.1
EOF

# Sign server certificate
openssl x509 -req -days $VALIDITY_DAYS -in "$CERT_DIR/server.csr" \
  -CA "$CERT_DIR/ca.crt" -CAkey "$CERT_DIR/ca-key.pem" -CAcreateserial \
  -out "$CERT_DIR/server.crt" -extfile "$CERT_DIR/server-extfile.cnf"

# Convert to standard names
cp "$CERT_DIR/server.crt" "$CERT_DIR/server.crt"
cp "$CERT_DIR/server-key.pem" "$CERT_DIR/server.key"

# Clean up
rm -f "$CERT_DIR/server.csr" "$CERT_DIR/server-extfile.cnf" "$CERT_DIR/ca.srl"

# Set appropriate permissions
chmod 644 "$CERT_DIR/ca.crt" "$CERT_DIR/server.crt"
chmod 600 "$CERT_DIR/server.key" "$CERT_DIR/ca-key.pem"

echo "TLS certificates generated successfully!"
echo "CA Certificate: $CERT_DIR/ca.crt"
echo "Server Certificate: $CERT_DIR/server.crt"
echo "Server Key: $CERT_DIR/server.key"