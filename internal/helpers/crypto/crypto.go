// Package crypto реализует гибридное шифрование тела запроса агента
// публичным ключом RSA и его расшифровку приватным ключом на сервере.
//
// RSA-OAEP с ключом 2048 бит шифрует не больше 190 байт, поэтому само тело
// шифруется AES-256-GCM случайным одноразовым ключом, а RSA-OAEP (SHA-256)
// шифрует только этот AES-ключ. Результат — один срез байт:
//
//	[RSA(aesKey)][nonce][AES-GCM(body) + тег]
//
// Длина первой части равна размеру RSA-ключа (256 байт для 2048 бит), nonce
// занимает 12 байт, остаток — шифротекст с 16-байтовым тегом целостности.
//
// Ключи хранятся в PEM-файлах в формате PKCS#1 ("RSA PUBLIC KEY" и
// "RSA PRIVATE KEY"), например:
//
//	openssl genrsa -traditional -out private.pem 2048
//	openssl rsa -in private.pem -RSAPublicKey_out -out public.pem
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
)

// ReadPublicKey читает публичный RSA-ключ из PEM-файла в формате PKCS#1.
//
// Возвращает ошибку, если у файла нет расширения .pem, файл не читается,
// в нём нет PEM-блока или ключ не разбирается как PKCS#1.
func ReadPublicKey(path string) (*rsa.PublicKey, error) {
	if !strings.HasSuffix(path, ".pem") {
		return nil, errors.New("incorrect certificate extension")
	}

	publicKeyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	publicKeyPemBlock, _ := pem.Decode(publicKeyBytes)
	if publicKeyPemBlock == nil {
		return nil, errors.New("public key: no PEM block found")
	}

	publicKey, err := x509.ParsePKCS1PublicKey(publicKeyPemBlock.Bytes)
	if err != nil {
		return nil, err
	}

	return publicKey, nil
}

// EncryptBodyWithKey шифрует message произвольной длины публичным ключом
// pubKey по гибридной схеме RSA-OAEP + AES-256-GCM (формат результата описан
// в документации пакета).
//
// Для каждого вызова генерируются новые AES-ключ и nonce, поэтому
// шифрование одних и тех же данных каждый раз даёт разный результат.
// Результат расшифровывается [DecryptBodyWithKey].
func EncryptBodyWithKey(message []byte, pubKey *rsa.PublicKey) ([]byte, error) {
	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, fmt.Errorf("error generate aes key %w", err)
	}

	aesBlock, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("error generate new cipher block %w", err)
	}

	aesgcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, fmt.Errorf("error generate new gcm %w", err)
	}

	nonce := make([]byte, aesgcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("error generate nonce %w", err)
	}

	cipherText := aesgcm.Seal(nil, nonce, message, nil)

	encryptKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, aesKey, nil)
	if err != nil {
		return nil, fmt.Errorf("error encryption aes key %w", err)
	}

	result := make([]byte, 0, len(encryptKey)+len(nonce)+len(cipherText))
	result = append(result, encryptKey...)
	result = append(result, nonce...)
	result = append(result, cipherText...)

	return result, nil
}

// ReadPrivateKey читает приватный RSA-ключ из PEM-файла в формате PKCS#1.
//
// Возвращает ошибку, если у файла нет расширения .pem, файл не читается,
// в нём нет PEM-блока или ключ не разбирается как PKCS#1.
func ReadPrivateKey(path string) (*rsa.PrivateKey, error) {
	if !strings.HasSuffix(path, ".pem") {
		return nil, errors.New("incorrect certificate extension")
	}

	privateKeyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	privateKeyPemBlock, _ := pem.Decode(privateKeyBytes)
	if privateKeyPemBlock == nil {
		return nil, errors.New("private key: no PEM block found")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyPemBlock.Bytes)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

// DecryptBodyWithKey расшифровывает body, полученный от [EncryptBodyWithKey],
// приватным ключом privateKey.
//
// Возвращает ошибку, если body короче минимального пакета, AES-ключ не
// расшифровывается этим приватным ключом или данные были изменены
// (не сошёлся тег GCM). Частично расшифрованные данные не возвращаются.
func DecryptBodyWithKey(body []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	keyLen := privateKey.Size()

	if len(body) < keyLen {
		return nil, errors.New("body is too short")
	}

	aesEncrypted, rest := body[:keyLen], body[keyLen:]

	aesKey, err := rsa.DecryptOAEP(sha256.New(), nil, privateKey, aesEncrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("error decryption aes key %w", err)
	}

	aesblock, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("error generate new cipher block %w", err)
	}

	aesgcm, err := cipher.NewGCM(aesblock)
	if err != nil {
		return nil, fmt.Errorf("error generate new gcm %w", err)
	}

	nc := aesgcm.NonceSize()
	oh := aesgcm.Overhead()

	if len(rest) < nc+oh {
		return nil, errors.New("body is too short")
	}

	nonce, cipherText := rest[:nc], rest[nc:]

	decryptBody, err := aesgcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, fmt.Errorf("error decrypt ciphered message %w", err)
	}

	return decryptBody, nil
}
