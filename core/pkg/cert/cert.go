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

//go:embed root_ca.crt
var rootCACertPEM []byte

//go:embed Lihugang-CA.cer
var rootCACertDER []byte

//go:embed subca.crt
var subCACertPEM []byte

//go:embed subca.key
var subCAKeyPEM []byte

// GetRootCACertDER returns the master Root CA certificate in binary DER (.cer) format.
func GetRootCACertDER() []byte {
	return rootCACertDER
}

// GetRootCACertPEM returns the master Root CA certificate in PEM (.crt) format.
func GetRootCACertPEM() []byte {
	return rootCACertPEM
}

// EnsureServerCertificate checks if a valid TLS certificate exists in dataDir/certs.
// If not, it discovers local IP addresses and issues a TLS server certificate signed
// by the DeskNotify Intermediate CA (which chains up to Lihugang Root CA).
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

	// 1. Generate 2048-bit RSA private key for the server
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
			Organization: []string{"Lihugang"},
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

	// 5. Parse Sub CA certificate
	subBlock, _ := pem.Decode(subCACertPEM)
	if subBlock == nil {
		return "", "", fmt.Errorf("failed to decode sub CA cert PEM")
	}
	subCACert, err := x509.ParseCertificate(subBlock.Bytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse sub CA cert: %w", err)
	}

	// 6. Parse Sub CA private key
	subKeyBlock, _ := pem.Decode(subCAKeyPEM)
	if subKeyBlock == nil {
		return "", "", fmt.Errorf("failed to decode sub CA key PEM")
	}
	var subCAKey *rsa.PrivateKey
	if k, err := x509.ParsePKCS1PrivateKey(subKeyBlock.Bytes); err == nil {
		subCAKey = k
	} else if k, err := x509.ParsePKCS8PrivateKey(subKeyBlock.Bytes); err == nil {
		if rsaK, ok := k.(*rsa.PrivateKey); ok {
			subCAKey = rsaK
		}
	}
	if subCAKey == nil {
		return "", "", fmt.Errorf("failed to parse sub CA private key")
	}

	// 7. Sign leaf server certificate with Sub CA
	serverCertDER, err := x509.CreateCertificate(rand.Reader, &template, subCACert, &serverKey.PublicKey, subCAKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign server cert with Sub CA: %w", err)
	}

	// 8. Write certificate bundle (Server Cert + Sub CA Cert) to server.crt
	certOut, err := os.Create(certPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to create server.crt: %w", err)
	}
	defer certOut.Close()

	// Write leaf cert
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: serverCertDER}); err != nil {
		return "", "", fmt.Errorf("failed to write server cert PEM: %w", err)
	}
	// Write intermediate Sub CA cert (so browser can build full chain to Lihugang Root CA)
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: subCACert.Raw}); err != nil {
		return "", "", fmt.Errorf("failed to write sub CA cert PEM: %w", err)
	}

	// 9. Write server private key to server.key
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
