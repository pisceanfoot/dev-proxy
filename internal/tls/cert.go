package tls

import (
	stdtls "crypto/tls"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"sync"
	"time"
)

// CertManager generates and caches self-signed certificates for a server.
type CertManager struct {
	mu       sync.RWMutex
	certCache map[string]*CertPair // key: "server" or file path prefix
}

// CertPair holds the certificate and private key as PEM bytes plus parsed leaf cert.
type CertPair struct {
	CertPEM []byte
	KeyPEM  []byte
	Leaf    *x509.Certificate
}

// NewCertManager creates a new CertManager with an empty cache.
func NewCertManager() *CertManager {
	return &CertManager{
		certCache: make(map[string]*CertPair),
	}
}

// GetOrGenerate returns the cached certificate for the given key, or generates
// a new one if not cached. Certificates are valid for 1 year.
func (cm *CertManager) GetOrGenerate(key string) (*CertPair, error) {
	cm.mu.RLock()
	if cp, ok := cm.certCache[key]; ok {
		cm.mu.RUnlock()
		return cp, nil
	}
	cm.mu.RUnlock()

	cp, err := generateSelfSigned()
	if err != nil {
		return nil, err
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()
	if existing, ok := cm.certCache[key]; ok {
		return existing, nil
	}
	cm.certCache[key] = cp
	return cp, nil
}

// LoadFromDisk loads a certificate pair from PEM files on disk and caches it.
func (cm *CertManager) LoadFromDisk(certFile, keyFile string) (*CertPair, error) {
	cacheKey := certFile + ":" + keyFile

	cm.mu.RLock()
	if cp, ok := cm.certCache[cacheKey]; ok {
		cm.mu.RUnlock()
		return cp, nil
	}
	cm.mu.RUnlock()

	cp, err := loadFromDisk(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()
	if existing, ok := cm.certCache[cacheKey]; ok {
		return existing, nil
	}
	cm.certCache[cacheKey] = cp
	return cp, nil
}

func loadFromDisk(certFile, keyFile string) (*CertPair, error) {
	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("certificate file not found: %s", certFile)
	}
	if _, err := os.Stat(keyFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("key file not found: %s", keyFile)
	}

	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("read certificate file %s: %w", certFile, err)
	}
	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("read key file %s: %w", keyFile, err)
	}

	// X509KeyPair validates that cert and key match — returns clear error if not.
	if _, err := stdtls.X509KeyPair(certPEM, keyPEM); err != nil {
		return nil, fmt.Errorf("certificate and key do not match (cert: %s, key: %s): %w", certFile, keyFile, err)
	}

	// Parse the leaf cert for the CertPair.Leaf field.
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM from %s", certFile)
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate from %s: %w", certFile, err)
	}

	return &CertPair{
		CertPEM: certPEM,
		KeyPEM:  keyPEM,
		Leaf:    leaf,
	}, nil
}

func generateSelfSigned() (*CertPair, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"dev-proxy"},
			CommonName:   "dev-proxy",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	leaf, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, err
	}

	return &CertPair{
		CertPEM: certPEM,
		KeyPEM:  keyPEM,
		Leaf:    leaf,
	}, nil
}
