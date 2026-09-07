package hash

import (
	"bytes"
	"testing"
)

// BenchmarkCreateHeaderHash измеряет стоимость вычисления HMAC-SHA256 подписи
// для типичного по размеру тела запроса.
func BenchmarkCreateHeaderHash(b *testing.B) {
	data := bytes.Repeat([]byte("x"), 1024)
	key := "secret-key"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = CreateHeaderHash(data, key)
	}
}
