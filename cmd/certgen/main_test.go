package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_WritesUsableCertificate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "certs") // директории ещё нет — run должен её создать
	const days = 30

	require.NoError(t, run([]string{"localhost", "127.0.0.1"}, dir, days))

	certPEM, err := os.ReadFile(filepath.Join(dir, tlsCertFile))
	require.NoError(t, err)
	keyPEM, err := os.ReadFile(filepath.Join(dir, tlsKeyFile))
	require.NoError(t, err)

	// Пара должна собираться в TLS-сертификат — так её загрузит сервер
	// (credentials.NewServerTLSFromFile).
	_, err = tls.X509KeyPair(certPEM, keyPEM)
	require.NoError(t, err, "сертификат и ключ должны составлять пару")

	block, _ := pem.Decode(certPEM)
	require.NotNil(t, block)
	assert.Equal(t, "CERTIFICATE", block.Type)

	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	// Агент кладёт этот сертификат в пул доверенных корней
	// (x509.CertPool.AppendCertsFromPEM в sender.createGRPCClient).
	assert.True(t, x509.NewCertPool().AppendCertsFromPEM(certPEM))
	assert.True(t, cert.IsCA)
	assert.Contains(t, cert.ExtKeyUsage, x509.ExtKeyUsageServerAuth)

	wantNotAfter := time.Now().AddDate(0, 0, days)
	assert.WithinDuration(t, wantNotAfter, cert.NotAfter, time.Hour, "срок действия задаётся -days")
}

// readCert возвращает сертификат, записанный run в dir.
func readCert(t *testing.T, dir string) *x509.Certificate {
	t.Helper()

	certPEM, err := os.ReadFile(filepath.Join(dir, tlsCertFile))
	require.NoError(t, err)

	block, _ := pem.Decode(certPEM)
	require.NotNil(t, block)

	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	return cert
}

func TestRun_SubjectAltNames(t *testing.T) {
	tests := []struct {
		name      string
		hosts     []string
		wantDNS   []string
		wantIPs   []string
		verifyFor []string // хосты, для которых сертификат должен проходить проверку
	}{
		{
			name:      "dns names and ip addresses",
			hosts:     []string{"localhost", "127.0.0.1", "myhost.example", "::1"},
			wantDNS:   []string{"localhost", "myhost.example"},
			wantIPs:   []string{"127.0.0.1", "::1"},
			verifyFor: []string{"localhost", "127.0.0.1", "myhost.example", "::1"},
		},
		{
			name:      "spaces around hosts are trimmed",
			hosts:     []string{"localhost", " 127.0.0.1", " myhost.example "},
			wantDNS:   []string{"localhost", "myhost.example"},
			wantIPs:   []string{"127.0.0.1"},
			verifyFor: []string{"myhost.example", "127.0.0.1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, run(tt.hosts, dir, 1))

			cert := readCert(t, dir)

			assert.Equal(t, tt.wantDNS, cert.DNSNames)

			gotIPs := make([]string, 0, len(cert.IPAddresses))
			for _, ip := range cert.IPAddresses {
				gotIPs = append(gotIPs, ip.String())
			}
			assert.Equal(t, tt.wantIPs, gotIPs)

			// Так же проверяет хост агент при TLS-рукопожатии.
			for _, host := range tt.verifyFor {
				assert.NoError(t, cert.VerifyHostname(host), "сертификат должен подходить для %q", host)
			}
			assert.Error(t, cert.VerifyHostname("other.example"), "для чужого хоста сертификат не подходит")
			assert.Error(t, cert.VerifyHostname("10.9.9.9"), "для чужого IP сертификат не подходит")
		})
	}
}

func TestRun_EmptyHost(t *testing.T) {
	tests := []struct {
		name  string
		hosts []string
	}{
		{name: "empty element", hosts: []string{"localhost", "", "127.0.0.1"}}, // -hosts "localhost,,127.0.0.1"
		{name: "spaces only", hosts: []string{"localhost", "  "}},              // -hosts "localhost,  "
		{name: "trailing comma", hosts: []string{"localhost", ""}},             // -hosts "localhost,"
		{name: "empty flag", hosts: []string{""}},                              // -hosts ""
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			err := run(tt.hosts, dir, 1)

			require.Error(t, err, "пустой хост не должен попадать в SAN")

			// При ошибке во входных данных файлы не создаются.
			_, statErr := os.Stat(filepath.Join(dir, tlsCertFile))
			assert.True(t, os.IsNotExist(statErr), "сертификат не должен быть записан")
			_, statErr = os.Stat(filepath.Join(dir, tlsKeyFile))
			assert.True(t, os.IsNotExist(statErr), "ключ не должен быть записан")
		})
	}
}
