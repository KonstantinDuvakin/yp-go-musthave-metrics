// Package models содержит общие для сервера и агента модели данных:
// метрику [Metrics] и событие аудита [Audit], а также допустимые
// типы метрик.
package models

// Типы метрик, поддерживаемые сервисом.
const (
	// Counter — счётчик: целочисленное значение, накапливаемое суммированием.
	Counter = "counter"
	// Gauge — измеритель: произвольное значение с плавающей точкой,
	// перезаписываемое при каждом обновлении.
	Gauge = "gauge"
)

// Metrics описывает одну метрику при обмене между агентом и сервером.
//
// Модель намеренно плоская, без вложенности. Delta и Value объявлены
// указателями, чтобы отличать явно переданный ноль от отсутствующего
// значения: незаданное поле не попадает в JSON благодаря omitempty.
// Для counter заполняется Delta, для gauge — Value.
type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // тип: [Counter] или [Gauge]
	Delta *int64   `json:"delta,omitempty"` // значение для counter
	Value *float64 `json:"value,omitempty"` // значение для gauge
	Hash  string   `json:"hash,omitempty"`  // подпись значения (опционально)
}

// Audit описывает событие аудита: факт приёма пакета метрик от клиента.
type Audit struct {
	TS        int64    `json:"ts"`         // отметка времени в миллисекундах
	Metrics   []string `json:"metrics"`    // имена принятых метрик
	IPAddress string   `json:"ip_address"` // адрес отправителя
}
