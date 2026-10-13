// Package tlstest выпускает самоподписанные TLS-сертификаты для тестов
// gRPC-связки агента и сервера.
//
// Сертификат действителен для localhost и 127.0.0.1 и записывается во
// временную директорию теста в тех же PEM-форматах, что выдаёт cmd/certgen.
package tlstest

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// WriteCert выпускает самоподписанный сертификат для localhost и 127.0.0.1,
// записывает сертификат и ключ в t.TempDir() и возвращает пути к файлам.
// При любой ошибке тест завершается через t.Fatal.
func WriteCert(t testing.TB) (certFile, keyFile string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("tlstest: generate key: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("tlstest: create certificate: %v", err)
	}

	dir := t.TempDir()
	certFile = filepath.Join(dir, "grpc-cert.pem")
	keyFile = filepath.Join(dir, "grpc-key.pem")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	if err = os.WriteFile(certFile, certPEM, 0o644); err != nil {
		t.Fatalf("tlstest: write certificate: %v", err)
	}
	if err = os.WriteFile(keyFile, keyPEM, 0o600); err != nil {
		t.Fatalf("tlstest: write key: %v", err)
	}

	return certFile, keyFile
}
