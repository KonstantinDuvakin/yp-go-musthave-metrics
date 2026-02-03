package sender

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

const serverURL = "http://localhost:8080/"

func SendMetrics(url string) {
	client := resty.New()
	_, err := client.R().SetHeader("Content-Type", "text/plain").Post(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
}

func UrlBuilder(metricType, name, value string) string {
	return fmt.Sprintf("%supdate/%s/%s/%s", serverURL, metricType, name, value)
}
