package config

import (
	"fmt"
	"time"
)

// intervalToSeconds разбирает строку длительности в формате
// [time.ParseDuration] (например, "1s", "500ms", "1m30s") и возвращает её
// в секундах, включая дробную часть.
//
// name — имя поля в конфиге, оно попадает в текст ошибки. Знак значения не
// проверяется: допустимый диапазон задают методы Validate конкретных
// конфигураций.
func intervalToSeconds(interval string, name string) (float64, error) {
	dur, err := time.ParseDuration(interval)

	if err != nil {
		return 0, fmt.Errorf("error parse duration from %s: %w", name, err)
	}

	seconds := dur.Seconds()

	return seconds, nil
}
