package collector

import "fmt"

// Collector собирает метрики рантайма Go. Конкретные значения зависят от
// среды выполнения, поэтому в примере проверяется лишь наличие ключей.
func ExampleCollector() {
	metrics := Collector()

	_, hasAlloc := metrics["Alloc"]
	_, hasRandom := metrics["RandomValue"]

	fmt.Println("есть Alloc:", hasAlloc)
	fmt.Println("есть RandomValue:", hasRandom)

	// Output:
	// есть Alloc: true
	// есть RandomValue: true
}
