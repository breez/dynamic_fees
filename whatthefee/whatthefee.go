package whatthefee

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type FeerateEstimation struct {
	Index   []int64   `json:"index"`
	Columns []string  `json:"columns"`
	Data    [][]int64 `json:"data"`
}

func GetFeerateEstimation(apiBaseUrl string) (*FeerateEstimation, error) {
	if apiBaseUrl == "" {
		return nil, fmt.Errorf("apiBaseUrl not set")
	}

	if !strings.HasSuffix(apiBaseUrl, "/") {
		apiBaseUrl = apiBaseUrl + "/"
	}
	req, err := http.NewRequestWithContext(
		context.Background(),
		"GET",
		apiBaseUrl+fmt.Sprintf("data.json?c=%v", time.Now().Unix()),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("http.NewRequestWithContext error: %w", err)
	}
	httpClient := http.DefaultClient
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("httpClient.Do error: %w", err)
	}

	defer resp.Body.Close()
	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		return nil, fmt.Errorf("error statuscode %v: %w", resp.StatusCode, err)
	}

	var fees FeerateEstimation
	err = json.NewDecoder(resp.Body).Decode(&fees)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &fees, nil
}
