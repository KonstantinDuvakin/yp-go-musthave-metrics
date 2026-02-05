package sender

import (
	"fmt"
	"net/http"
)

const serverURL = "http://localhost:8080/"

func SendMetrics(url string) {
	resp, err := http.Post(url, "text/plain", nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
}

func URLBuilder(metricType, name, value string) string {
	return fmt.Sprintf("%supdate/%s/%s/%s", serverURL, metricType, name, value)
}
