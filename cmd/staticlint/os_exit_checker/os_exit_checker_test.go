package osexitchecker

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestOsExitChecker(t *testing.T) {
	// analysistest.TestData() -> путь к папке testdata рядом с тестом.
	// "pkg1" — пакет внутри testdata, который прогоняем через анализатор.
	// Ожидаемые диагностики помечены комментариями // want в pkg1.go.
	analysistest.Run(t, analysistest.TestData(), OsExitChecker, "pkg1")
}
