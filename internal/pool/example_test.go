package pool_test

import (
	"bytes"
	"fmt"

	"github.com/KonstantinDuvakin/yp-go-musthave-metrics/internal/pool"
)

// Pool переиспользует буферы: перед возвратом в пул каждый буфер очищается.
func ExamplePool() {
	buffers := pool.New(func() *bytes.Buffer { return &bytes.Buffer{} })

	buf := buffers.Get()
	buf.WriteString("hello")
	fmt.Println(buf.Len())

	buffers.Put(buf) // вызовет buf.Reset()

	buf = buffers.Get()
	fmt.Println(buf.Len())

	// Output:
	// 5
	// 0
}
