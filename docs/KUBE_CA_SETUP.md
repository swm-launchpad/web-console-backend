# Kubernetes CA Certificate Setup

## Why is this required?

The application connects to the Kubernetes API server over HTTPS. To verify the server's identity and establish a secure connection, the application needs the Certificate Authority (CA) certificate that signed the API server's certificate.

## How to Download CA Certificate

Extract the CA certificate from your kubeconfig file using kubectl:

```bash
# Extract CA certificate from kubeconfig
kubectl config view --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}' | base64 -d > /etc/kube/ca.crt

# Verify the certificate
openssl x509 -in /etc/kube/ca.crt -text -noout | grep -A2 "Issuer:\|Subject:"
```

Expected output:
```
Issuer: CN = kubernetes
Subject: CN = kubernetes
```

## Configure the Application

Set the CA certificate path in `.env`:

```bash
KUBE_CA_CERT_PATH=/etc/kube/ca.crt
```

## Verify

Test the connection:

```bash
curl --cacert /etc/kube/ca.crt \
  -H "Authorization: Bearer ${KUBE_SERVICE_ACCOUNT_TOKEN}" \
  "${KUBE_API_SERVER}/api/v1/namespaces/default"
```

If successful, the CA certificate is correctly configured.

## Security Notes

- **Production**: Always set `KUBE_CA_CERT_PATH` for secure TLS verification
- **Development**: You may temporarily use `KUBE_INSECURE_SKIP_TLS_VERIFY=true` (not recommended)
