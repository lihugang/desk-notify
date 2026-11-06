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

	// Read and verify certificate
	data, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("failed to read cert file: %v", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatalf("no PEM data found in cert file")
	}

	serverCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse server cert: %v", err)
	}

	if serverCert.Subject.CommonName != "传纸条 Web Server (DeskNotify)" {
		t.Errorf("unexpected CommonName: %v", serverCert.Subject.CommonName)
	}

	t.Logf("Server certificate successfully generated! CN: %s, IPs: %v, DNS: %v",
		serverCert.Subject.CommonName, serverCert.IPAddresses, serverCert.DNSNames)
}
