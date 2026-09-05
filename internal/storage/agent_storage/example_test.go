package agentstorage

import "fmt"

// Накопление метрик агентом и получение согласованного снимка через Snapshot.
func ExampleAgentStorage() {
	s := NewAgentStorage()

	s.SetGauge("Alloc", 42.5)
	s.AddCounter("PollCount")
	s.AddCounter("PollCount")

	gauges, counters := s.Snapshot()

	fmt.Println("Alloc:", gauges["Alloc"])
	fmt.Println("PollCount:", counters["PollCount"])

	// Output:
	// Alloc: 42.5
	// PollCount: 2
}
