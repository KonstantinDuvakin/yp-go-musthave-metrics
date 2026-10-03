// Команда keygen генерирует пару RSA-ключей для шифрования тела запросов
// агента (см. пакет internal/helpers/crypto).
//
// Ключи записываются в PEM-файлы в формате PKCS#1: private.pem
// ("RSA PRIVATE KEY", права 0600) и public.pem ("RSA PUBLIC KEY", права
// 0644). Публичный ключ передаётся агенту, приватный — серверу (флаг
// -crypto-key или переменная CRYPTO_KEY). Запускать из корня модуля:
//
//	go run ./cmd/keygen
//	go run ./cmd/keygen -dir ./keys -bits 4096
//
// Флаги:
//   - -dir — директория для ключей (по умолчанию текущая), создаётся при
//     необходимости;
//   - -bits — размер ключа в битах, не меньше 2048 (по умолчанию 2048);
//   - -force — перезаписать существующие файлы ключей.
//
// Без -force команда завершается с ошибкой, если хотя бы один из файлов уже
// существует, и не изменяет ни одного из них.
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

const (
	privateKeyFile = "private.pem"
	publicKeyFile  = "public.pem"

	// minBits — минимальный размер ключа: с меньшим RSA-OAEP (SHA-256) не
	// вместит AES-ключ гибридной схемы с запасом и считается небезопасным.
	minBits = 2048
)

func main() {
	dir := flag.String("dir", ".", "директория для файлов ключей")
	bits := flag.Int("bits", minBits, "размер RSA-ключа в битах")
	force := flag.Bool("force", false, "перезаписать существующие файлы ключей")
	flag.Parse()

	if err := run(*dir, *bits, *force); err != nil {
		log.Fatal("keygen: ", err)
	}

	fmt.Printf("Keys written to %s and %s\n",
		filepath.Join(*dir, privateKeyFile), filepath.Join(*dir, publicKeyFile))
}

// run генерирует пару RSA-ключей размером bits и записывает её в dir.
// Если force равен false и любой из файлов уже существует, возвращает
// ошибку, ничего не записывая.
func run(dir string, bits int, force bool) error {
	if bits < minBits {
		return fmt.Errorf("key size %d is too small, minimum is %d", bits, minBits)
	}

	privatePath := filepath.Join(dir, privateKeyFile)
	publicPath := filepath.Join(dir, publicKeyFile)

	if !force {
		for _, path := range []string{privatePath, publicPath} {
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("%s already exists, use -force to overwrite", path)
			} else if !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		}
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	publicPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey),
	})

	if err = os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	if err = os.WriteFile(privatePath, privatePEM, 0o600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}

	if err = os.WriteFile(publicPath, publicPEM, 0o644); err != nil {
		return fmt.Errorf("write public key: %w", err)
	}

	return nil
}
