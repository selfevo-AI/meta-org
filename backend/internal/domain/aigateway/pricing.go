package aigateway

import "math"

type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type Price struct {
	InputPer1K  float64 `json:"input_price_per_1k"`
	OutputPer1K float64 `json:"output_price_per_1k"`
}

func CalculateCost(usage TokenUsage, price Price) float64 {
	cost := (float64(usage.InputTokens)/1000.0)*price.InputPer1K + (float64(usage.OutputTokens)/1000.0)*price.OutputPer1K
	return math.Round(cost*1e8) / 1e8
}
