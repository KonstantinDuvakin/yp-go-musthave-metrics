package main

import (
	"os"
	xos "os"
)

// fakeOS — «тень»: чужой тип с методом Exit, не имеющий отношения к пакету os.
type fakeOS struct{}

func (fakeOS) Exit(code int) {}

func run(f func()) { f() }

// helper вызывает os.Exit ВНЕ функции main — диагностики быть не должно.
func helper() {
	os.Exit(1)
}

func main() {
	os.Exit(1) // want "can't call os.Exit in main"

	xos.Exit(2) // want "can't call os.Exit in main"

	// вложенный вызов внутри литерала-аргумента — тоже должен ловиться
	run(func() {
		os.Exit(3) // want "can't call os.Exit in main"
	})

	// тень: os здесь — локальная переменная чужого типа, не пакет os.
	{
		var os = fakeOS{}
		os.Exit(4)
		_ = os
	}
}
