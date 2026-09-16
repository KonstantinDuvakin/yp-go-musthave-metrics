package buildinfo

import "os"

// Значения обычно приходят из переменных пакета main, заполненных линкером
// при сборке с -ldflags "-X main.buildVersion=v1.0.0 ...".
func ExamplePrintBuildInfo() {
	PrintBuildInfo(os.Stdout, "v1.0.0", "2026/09/14 15:00:00", "98a9d3e")

	// Output:
	// Build version: v1.0.0
	// Build date: 2026/09/14 15:00:00
	// Build commit: 98a9d3e
}

// Если бинарник собран без -ldflags, переменные остаются пустыми и вместо
// них выводится N/A.
func ExamplePrintBuildInfo_notSet() {
	PrintBuildInfo(os.Stdout, "", "", "")

	// Output:
	// Build version: N/A
	// Build date: N/A
	// Build commit: N/A
}
