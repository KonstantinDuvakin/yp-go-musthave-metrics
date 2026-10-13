// Команда certgen выпускает самоподписанный TLS-сертификат для gRPC-связки
// агента и сервера.
//
// В директорию записываются два PEM-файла: grpc-cert.pem (сертификат,
// "CERTIFICATE") и grpc-key.pem (приватный ключ RSA в формате PKCS#1,
// "RSA PRIVATE KEY"). Сертификат и ключ передаются серверу (флаги
// -grpc-cert и -grpc-key или переменные GRPC_CERT и GRPC_KEY), сертификат —
// агенту (флаг -grpc-cert или GRPC_CERT): агент доверяет только ему. Запускать
// из корня модуля:
//
//	go run ./cmd/certgen
//	go run ./cmd/certgen -dir ./keys -hosts localhost,127.0.0.1 -days 365
//
// Флаги:
//   - -hosts — DNS-имена и IP-адреса через запятую, для которых действителен
//     сертификат (по умолчанию localhost,127.0.0.1); пробелы вокруг элементов
//     отбрасываются, IP-адреса попадают в SAN как IP, остальное — как DNS-имена.
//     Пустой элемент (например, "localhost,," или запятая в конце) — ошибка.
//     Агент сверяет с ними хост из своего -grpc-address;
//   - -days — срок действия сертификата в днях (по умолчанию 365);
//   - -dir — директория для файлов (по умолчанию текущая), создаётся при
//     необходимости.
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	tlsCertFile = "grpc-cert.pem" // сертификат: серверу и агенту
	tlsKeyFile  = "grpc-key.pem"  // приватный ключ: только серверу
	bits        = 2048            // размер RSA-ключа сертификата
)

func main() {
	hostsString := flag.String("hosts", "localhost,127.0.0.1", "имена и IP через запятую, для которых действителен сертификат")
	days := flag.Int("days", 365, "срок действия сертификата в днях")
	dir := flag.String("dir", ".", "директория для файлов сертификата")
	flag.Parse()

	hosts := strings.Split(*hostsString, ",")

	if err := run(hosts, *dir, *days); err != nil {
		log.Fatal("certgen error: ", err)
	}

	fmt.Printf("cert written to %s and %s\n",
		filepath.Join(*dir, tlsCertFile), filepath.Join(*dir, tlsKeyFile))
}

// run генерирует RSA-ключ и самоподписанный сертификат для hosts (DNS-имена
// и IP-адреса, пробелы вокруг обрезаются) сроком на days дней и записывает
// их в dir, создавая директорию при необходимости. Существующие файлы
// перезаписываются.
//
// Если какой-то элемент hosts после обрезки пробелов пуст, возвращает ошибку
// и ничего не записывает: пустое имя в SAN недопустимо.
func run(hosts []string, dir string, days int) error {
	tlsCertFilePath := filepath.Join(dir, tlsCertFile)
	tlsKeyFilePath := filepath.Join(dir, tlsKeyFile)

	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return fmt.Errorf("error generate key: %w", err)
	}

	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization: []string{"Konstantin_Duvakin"},
			Country:      []string{"RU"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, days),
		SubjectKeyId:          []byte{1, 2, 3, 4, 6},
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}

	for _, h := range hosts {
		hTrimmed := strings.TrimSpace(h)
		if hTrimmed == "" {
			return errors.New("empty host has been passed")
		}
		if ip := net.ParseIP(hTrimmed); ip != nil {
			cert.IPAddresses = append(cert.IPAddresses, ip)
		} else {
			cert.DNSNames = append(cert.DNSNames, hTrimmed)
		}
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	if err != nil {
		return fmt.Errorf("error create cert: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	if err = os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	if err = os.WriteFile(tlsCertFilePath, certPEM, 0o644); err != nil {
		return fmt.Errorf("ошибка записи сертификата: %w", err)
	}

	if err = os.WriteFile(tlsKeyFilePath, keyPEM, 0o600); err != nil {
		return fmt.Errorf("ошибка записи ключа: %w", err)
	}

	return nil
}
