package cert

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
)

func TestEnsureServerCertificate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cert-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	certPath, keyPath, err := EnsureServerCertificate(tempDir)
	if err != nil {
		t.Fatalf("EnsureServerCertificate failed: %v", err)
	}

	if _, err := os.Stat(certPath); err != nil {
		t.Errorf("certPath does not exist: %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("keyPath does not exist: %v", err)
	}

	// Read and verify certificate chain
	data, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("failed to read cert file: %v", err)
	}

	// 1. Decode Leaf Cert
	block, rest := pem.Decode(data)
	if block == nil {
		t.Fatalf("no leaf PEM data found in cert file")
	}
	leafCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse leaf server cert: %v", err)
	}

	// 2. Decode Intermediate Sub CA Cert
	subBlock, _ := pem.Decode(rest)
	if subBlock == nil {
		t.Fatalf("no intermediate PEM data found in cert file")
	}
	subCert, err := x509.ParseCertificate(subBlock.Bytes)
	if err != nil {
		t.Fatalf("failed to parse intermediate cert: %v", err)
	}

	// 3. Parse Root CA
	rootBlock, _ := pem.Decode(rootCACertPEM)
	rootCert, err := x509.ParseCertificate(rootBlock.Bytes)
	if err != nil {
		t.Fatalf("failed to parse root CA cert: %v", err)
	}

	roots := x509.NewCertPool()
	roots.AddCert(rootCert)

	intermediates := x509.NewCertPool()
	intermediates.AddCert(subCert)

	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		DNSName:       "localhost",
	}

	chains, err := leafCert.Verify(opts)
	if err != nil {
		t.Fatalf("chain verification failed against Lihugang Root CA: %v", err)
	}

	t.Logf("Server certificate successfully verified against Lihugang Root CA via Sub CA! Chains: %d, IPs: %v",
		len(chains), leafCert.IPAddresses)
}
