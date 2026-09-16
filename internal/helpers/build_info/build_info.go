// Package buildinfo выводит сведения о сборке приложения: версию, дату и
// коммит. Значения подставляются линкером на этапе компиляции через
// -ldflags "-X main.buildVersion=..." и передаются сюда из пакета main.
package buildinfo

import (
	"fmt"
	"io"
)

// NA выводится вместо значения, которое не было задано при сборке.
const NA = "N/A"

// PrintBuildInfo выводит сведения о сборке в w в формате:
//
//	Build version: <version>
//	Build date: <date>
//	Build commit: <commit>
//
// Пустые значения заменяются на [NA].
func PrintBuildInfo(w io.Writer, version, date, commit string) {
	fmt.Fprintf(w, "Build version: %s\n", valueOrNA(version))
	fmt.Fprintf(w, "Build date: %s\n", valueOrNA(date))
	fmt.Fprintf(w, "Build commit: %s\n", valueOrNA(commit))
}

func valueOrNA(s string) string {
	if s == "" {
		return NA
	}
	return s
}
