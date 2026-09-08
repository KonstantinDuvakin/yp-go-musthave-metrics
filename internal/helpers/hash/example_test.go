package hash

import "fmt"

// Вычисление HMAC-SHA256 подписи для тела запроса. Значение помещается в
// заголовок HashSHA256 и позволяет серверу проверить целостность данных.
func ExampleCreateHeaderHash() {
	data := []byte(`{"id":"Alloc","type":"gauge","value":42.5}`)

	sign := CreateHeaderHash(data, "secret-key")
	fmt.Println(sign)

	// Output:
	// 6296720a91e3f250acd382f6142a11e26aead9677215e8025cfeece2f5b06594
}

// Подпись детерминирована: одинаковые данные и ключ дают одинаковый результат,
// а другой ключ — другой. На этом строится проверка подписи на сервере.
func ExampleCreateHeaderHash_verify() {
	data := []byte("payload")

	a := CreateHeaderHash(data, "key-1")
	b := CreateHeaderHash(data, "key-1")
	c := CreateHeaderHash(data, "key-2")

	fmt.Println("совпадает с тем же ключом:", a == b)
	fmt.Println("совпадает с другим ключом:", a == c)

	// Output:
	// совпадает с тем же ключом: true
	// совпадает с другим ключом: false
}
