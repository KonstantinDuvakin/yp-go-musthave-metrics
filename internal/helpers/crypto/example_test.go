package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
)

// Агент шифрует тело публичным ключом, сервер расшифровывает приватным.
// В реальном коде ключи читаются из файлов через [ReadPublicKey] и
// [ReadPrivateKey]; здесь пара генерируется в памяти.
func ExampleEncryptBodyWithKey() {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}

	body := []byte(`[{"id":"Alloc","type":"gauge","value":42.5}]`)

	encrypted, err := EncryptBodyWithKey(body, &privateKey.PublicKey)
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	// 256 (RSA-2048) + 12 (nonce) + len(body) + 16 (тег GCM)
	fmt.Println("размер пакета:", len(encrypted))

	decrypted, err := DecryptBodyWithKey(encrypted, privateKey)
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}
	fmt.Println("расшифровано:", string(decrypted))

	// Output:
	// размер пакета: 328
	// расшифровано: [{"id":"Alloc","type":"gauge","value":42.5}]
}

// Изменение хотя бы одного байта пакета обнаруживается при расшифровке.
func ExampleDecryptBodyWithKey_tampered() {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}

	encrypted, err := EncryptBodyWithKey([]byte("payload"), &privateKey.PublicKey)
	if err != nil {
		fmt.Println("ошибка:", err)
		return
	}

	encrypted[len(encrypted)-1] ^= 0x01

	_, err = DecryptBodyWithKey(encrypted, privateKey)
	fmt.Println("ошибка при подмене:", err != nil)

	// Output:
	// ошибка при подмене: true
}
