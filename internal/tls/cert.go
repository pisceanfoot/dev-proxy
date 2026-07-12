package tls

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"sync"
	"time"
)

// CertManager generates and caches self-signed certificates per-route.
type CertManager struct {
	mu       sync.RWMutex
	certCache map[string]*CertPair // key: route identifier (e.g., ":8443")
}

// CertPair holds the certificate and private key as PEM bytes.
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
	// Double-check after acquiring write lock (another goroutine may have populated it)
	if existing, ok := cm.certCache[key]; ok {
		return existing, nil
	}
	cm.certCache[key] = cp
	return cp, nil
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
