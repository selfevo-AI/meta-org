package aigateway

import "testing"

func TestCalculateCost(t *testing.T) {
	cost := CalculateCost(TokenUsage{InputTokens: 1200, OutputTokens: 300}, Price{InputPer1K: 0.01, OutputPer1K: 0.03})
	if cost != 0.021 {
		t.Fatalf("cost = %.6f, want 0.021", cost)
	}
}
