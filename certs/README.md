# SSL Certificates

This directory contains SSL/TLS certificates for the server.
To generate certs automatically use run the script in the scripts folder and follow the instructions.

## Required Files
- `server.crt` - SSL certificate
- `server.key` - Private key

## Development Certificates
For local development, generate self-signed certificates:

```bash
# Generate private key
openssl genrsa -out server.key 2048

# Generate CSR
openssl req -new -key server.key -out server.csr

# Generate certificate
openssl x509 -req -days 365 -in server.csr -signkey server.key -out server.crt
```

## Security Notes
- Never commit certificate files to version control
- Keep private keys secure
- Regularly rotate certificates
- Use appropriate permissions (600 for private keys)