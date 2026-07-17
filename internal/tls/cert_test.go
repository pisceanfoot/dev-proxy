package tls

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewCertManager(t *testing.T) {
	cm := NewCertManager()
	if cm == nil {
		t.Fatal("expected CertManager, got nil")
	}
	if cm.certCache == nil {
		t.Fatal("expected certCache to be initialized")
	}
}

func TestGetOrGenerate_CacheMiss(t *testing.T) {
	cm := NewCertManager()
	cp, err := cm.GetOrGenerate("test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cp == nil {
		t.Fatal("expected CertPair, got nil")
	}
	if len(cp.CertPEM) == 0 {
		t.Fatal("expected non-empty cert PEM")
	}
	if len(cp.KeyPEM) == 0 {
		t.Fatal("expected non-empty key PEM")
	}
	if cp.Leaf == nil {
		t.Fatal("expected parsed leaf certificate")
	}
}

func TestGetOrGenerate_CacheHit(t *testing.T) {
	cm := NewCertManager()
	cp1, err := cm.GetOrGenerate("test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cp2, err := cm.GetOrGenerate("test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be the same cached object
	if cp1 != cp2 {
		t.Fatal("expected cached CertPair to be returned on second call")
	}
}

func TestGetOrGenerate_Concurrent(t *testing.T) {
	cm := NewCertManager()
	var wg sync.WaitGroup
	results := make([]*CertPair, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cp, err := cm.GetOrGenerate("concurrent-key")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			results[idx] = cp
		}(i)
	}

	wg.Wait()

	// All should return the same cached object
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			t.Fatal("expected all concurrent calls to return the same CertPair")
		}
	}
}

func TestLoadFromDisk_Success(t *testing.T) {
	// Generate a self-signed cert and write to temp files
	cp, err := generateSelfSigned()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	if err := os.WriteFile(certFile, cp.CertPEM, 0644); err != nil {
		t.Fatalf("failed to write cert file: %v", err)
	}
	if err := os.WriteFile(keyFile, cp.KeyPEM, 0600); err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	cm := NewCertManager()
	loaded, err := cm.LoadFromDisk(certFile, keyFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected CertPair, got nil")
	}
	if string(loaded.CertPEM) != string(cp.CertPEM) {
		t.Fatal("expected cert PEM to match")
	}
	if loaded.Leaf == nil {
		t.Fatal("expected parsed leaf certificate")
	}

	// Second call should use cache
	cached, err := cm.LoadFromDisk(certFile, keyFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cached != loaded {
		t.Fatal("expected cached CertPair on second call")
	}
}

func TestLoadFromDisk_MissingCertFile(t *testing.T) {
	cm := NewCertManager()
	_, err := cm.LoadFromDisk("/nonexistent/cert.pem", "/nonexistent/key.pem")
	if err == nil {
		t.Fatal("expected error for missing cert file")
	}
	if !os.IsNotExist(err) {
		// The error should mention the cert file
		errStr := err.Error()
		if !contains(errStr, "certificate file not found") {
			t.Fatalf("expected 'certificate file not found' error, got: %v", err)
		}
	}
}

func TestLoadFromDisk_MissingKeyFile(t *testing.T) {
	// Create a cert file but no key file
	cp, err := generateSelfSigned()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")

	if err := os.WriteFile(certFile, cp.CertPEM, 0644); err != nil {
		t.Fatalf("failed to write cert file: %v", err)
	}

	cm := NewCertManager()
	_, err = cm.LoadFromDisk(certFile, filepath.Join(tmpDir, "missing.key"))
	if err == nil {
		t.Fatal("expected error for missing key file")
	}
	errStr := err.Error()
	if !contains(errStr, "key file not found") {
		t.Fatalf("expected 'key file not found' error, got: %v", err)
	}
}

func TestLoadFromDisk_MismatchedCertAndKey(t *testing.T) {
	// Generate two different cert/key pairs
	cp1, err := generateSelfSigned()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cp2, err := generateSelfSigned()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	// Write cert from pair 1 and key from pair 2
	if err := os.WriteFile(certFile, cp1.CertPEM, 0644); err != nil {
		t.Fatalf("failed to write cert file: %v", err)
	}
	if err := os.WriteFile(keyFile, cp2.KeyPEM, 0600); err != nil {
		t.Fatalf("failed to write key file: %v", err)
	}

	cm := NewCertManager()
	_, err = cm.LoadFromDisk(certFile, keyFile)
	if err == nil {
		t.Fatal("expected error for mismatched cert and key")
	}
	errStr := err.Error()
	if !contains(errStr, "certificate and key do not match") {
		t.Fatalf("expected mismatch error, got: %v", err)
	}
}

func TestGenerateSelfSigned(t *testing.T) {
	cp, err := generateSelfSigned()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify cert PEM structure
	certBlock, _ := pem.Decode(cp.CertPEM)
	if certBlock == nil {
		t.Fatal("failed to decode certificate PEM")
	}
	if certBlock.Type != "CERTIFICATE" {
		t.Fatalf("expected CERTIFICATE block, got %q", certBlock.Type)
	}

	// Verify key PEM structure
	keyBlock, _ := pem.Decode(cp.KeyPEM)
	if keyBlock == nil {
		t.Fatal("failed to decode key PEM")
	}
	if keyBlock.Type != "EC PRIVATE KEY" {
		t.Fatalf("expected EC PRIVATE KEY block, got %q", keyBlock.Type)
	}

	// Verify leaf certificate
	if cp.Leaf == nil {
		t.Fatal("expected parsed leaf certificate")
	}

	// Verify it's a valid, self-signed certificate
	if cp.Leaf.Subject.CommonName != "dev-proxy" {
		t.Fatalf("expected CommonName 'dev-proxy', got %q", cp.Leaf.Subject.CommonName)
	}

	// Verify key usage
	if cp.Leaf.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		t.Fatal("expected DigitalSignature key usage")
	}

	// Verify ExtKeyUsage includes ServerAuth
	foundServerAuth := false
	for _, usage := range cp.Leaf.ExtKeyUsage {
		if usage == x509.ExtKeyUsageServerAuth {
			foundServerAuth = true
			break
		}
	}
	if !foundServerAuth {
		t.Fatal("expected ServerAuth extended key usage")
	}

	// Verify it's valid (NotBefore <= now <= NotAfter)
	now := time.Now()
	if now.Before(cp.Leaf.NotBefore) {
		t.Fatal("certificate is not yet valid")
	}
	if now.After(cp.Leaf.NotAfter) {
		t.Fatal("certificate has expired")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
