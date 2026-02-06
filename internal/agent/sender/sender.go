package sender

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

func SendMetrics(url string) {
	client := resty.New()
	_, err := client.R().SetHeader("Content-Type", "text/plain").Post(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
}

func URLBuilder(address, metricType, name, value string) string {
	return fmt.Sprintf("http://%s/update/%s/%s/%s", address, metricType, name, value)
}
