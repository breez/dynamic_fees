package mempoolspace

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type RecommendedFees struct {
	FastestFee  uint `json:"fastestFee"`
	HalfHourFee uint `json:"halfHourFee"`
	HourFee     uint `json:"hourFee"`
	EconomyFee  uint `json:"economyFee"`
	MinimumFee  uint `json:"minimumFee"`
}

func GetRecommendedFees(apiBaseUrl string) (*RecommendedFees, error) {
	if apiBaseUrl == "" {
		return nil, fmt.Errorf("apiBaseUrl not set")
	}

	if !strings.HasSuffix(apiBaseUrl, "/") {
		apiBaseUrl = apiBaseUrl + "/"
	}
	req, err := http.NewRequestWithContext(
		context.Background(),
		"GET",
		apiBaseUrl+"fees/recommended",
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

	var fees RecommendedFees
	err = json.NewDecoder(resp.Body).Decode(&fees)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return &fees, nil
}
