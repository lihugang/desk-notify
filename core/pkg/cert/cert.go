package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

//go:embed ca.crt
var caCertPEM []byte

//go:embed DeskNotify-CA.cer
var caCertDER []byte

// GetRootCACertDER returns the Root CA certificate in binary DER (.cer) format.
func GetRootCACertDER() []byte {
	return caCertDER
}

// GetRootCACertPEM returns the Root CA certificate in PEM (.crt) format.
func GetRootCACertPEM() []byte {
	return caCertPEM
}

// EnsureServerCertificate checks if a valid TLS certificate exists in dataDir/certs.
// If not, it discovers local IP addresses and generates a TLS certificate
// (signed by DeskNotify Root CA if CA key is available, or self-signed with matching CA metadata).
func EnsureServerCertificate(dataDir string) (string, string, error) {
	certDir := filepath.Join(dataDir, "certs")
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create certs dir: %w", err)
	}

	certPath := filepath.Join(certDir, "server.crt")
	keyPath := filepath.Join(certDir, "server.key")

	// If both files already exist, return them directly
	if _, err1 := os.Stat(certPath); err1 == nil {
		if _, err2 := os.Stat(keyPath); err2 == nil {
			return certPath, keyPath, nil
		}
	}

	// 1. Generate 2048-bit RSA private key for server
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate server private key: %w", err)
	}

	// 2. Discover local IP addresses and hostnames
	ipList := []net.IP{net.ParseIP("127.0.0.1"), net.IPv6loopback}
	dnsList := []string{"localhost", "*.local"}

	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		dnsList = append(dnsList, hostname)
	}

	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				if ip4 := ipNet.IP.To4(); ip4 != nil && !ip4.IsLoopback() {
					ipList = append(ipList, ip4)
				}
			}
		}
	}

	// 3. Generate random serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate serial number: %w", err)
	}

	// 4. Create TLS Server Certificate Template (valid for 10 years)
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Country:      []string{"CN"},
			Organization: []string{"DeskNotify (传纸条)"},
			CommonName:   "传纸条 Web Server (DeskNotify)",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           ipList,
		DNSNames:              dnsList,
	}

	// 5. Check if CA private key is available on disk or env
	var caKey *rsa.PrivateKey
	var caCert *x509.Certificate
	caKeyCandidatePaths := []string{
		filepath.Join(dataDir, "ca.key"),
		filepath.Join(dataDir, "certs", "ca.key"),
		"certs/ca.key",
	}

	for _, p := range caKeyCandidatePaths {
		if data, err := os.ReadFile(p); err == nil {
			block, _ := pem.Decode(data)
			if block != nil {
				if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
					caKey = k
					break
				} else if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
					if rsaK, ok := k.(*rsa.PrivateKey); ok {
						caKey = rsaK
						break
					}
				}
			}
		}
	}

	var serverCertDER []byte
	var isCASigned bool

	if caKey != nil && len(caCertPEM) > 0 {
		caBlock, _ := pem.Decode(caCertPEM)
		if caBlock != nil {
			if c, err := x509.ParseCertificate(caBlock.Bytes); err == nil {
				caCert = c
				// Sign with CA private key
				der, err := x509.CreateCertificate(rand.Reader, &template, caCert, &serverKey.PublicKey, caKey)
				if err == nil {
					serverCertDER = der
					isCASigned = true
				}
			}
		}
	}

	if !isCASigned {
		// Self-signed certificate using server's own key
		template.Issuer = template.Subject
		der, err := x509.CreateCertificate(rand.Reader, &template, &template, &serverKey.PublicKey, serverKey)
		if err != nil {
			return "", "", fmt.Errorf("failed to create self-signed server cert: %w", err)
		}
		serverCertDER = der
	}

	// 6. Write certificate to server.crt
	certOut, err := os.Create(certPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create server.crt: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: serverCertDER}); err != nil {
		return "", "", fmt.Errorf("failed to write server cert PEM: %w", err)
	}

	// If CA signed, append Root CA certificate to bundle
	if isCASigned && caCert != nil {
		if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: caCert.Raw}); err != nil {
			return "", "", fmt.Errorf("failed to write CA cert PEM: %w", err)
		}
	}

	// 7. Write private key to server.key
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", "", fmt.Errorf("failed to create server.key: %w", err)
	}
	defer keyOut.Close()

	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(serverKey)}); err != nil {
		return "", "", fmt.Errorf("failed to write server key PEM: %w", err)
	}

	return certPath, keyPath, nil
}
