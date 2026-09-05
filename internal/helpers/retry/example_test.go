package retry

import (
	"context"
	"errors"
	"fmt"
)

// Do выполняет операцию и при временной ошибке повторяет её. Здесь операция
// завершается успешно с первой попытки, повтор не требуется.
func ExampleDo() {
	attempts := 0

	err := Do(context.Background(), IsHTTPRetriable, func() error {
		attempts++
		return nil
	})

	fmt.Println("попыток:", attempts)
	fmt.Println("ошибка:", err)

	// Output:
	// попыток: 1
	// ошибка: <nil>
}

// Если ошибка не признана временной, Do возвращает её сразу, без повторов.
func ExampleDo_nonRetriable() {
	attempts := 0

	// isRetriable всегда возвращает false — повторов не будет.
	notRetriable := func(error) bool { return false }

	err := Do(context.Background(), notRetriable, func() error {
		attempts++
		return errors.New("постоянная ошибка")
	})

	fmt.Println("попыток:", attempts)
	fmt.Println("ошибка:", err)

	// Output:
	// попыток: 1
	// ошибка: постоянная ошибка
}

// IsHTTPRetriable распознаёт временные сетевые ошибки. Обычная ошибка и nil
// временными не считаются.
func ExampleIsHTTPRetriable() {
	fmt.Println(IsHTTPRetriable(nil))
	fmt.Println(IsHTTPRetriable(errors.New("bad request")))

	// Output:
	// false
	// false
}
