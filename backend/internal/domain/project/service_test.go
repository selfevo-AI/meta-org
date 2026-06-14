package project

import (
	"testing"

	"github.com/google/uuid"
)

func TestPrepareAIUsageCostEntryInputRequiresSourceID(t *testing.T) {
	input := CreateCostEntryInput{Amount: 1.25}
	if err := prepareAIUsageCostEntryInput(&input); err == nil {
		t.Fatalf("prepareAIUsageCostEntryInput accepted missing source_id")
	}
}

func TestPrepareAIUsageCostEntryInputNormalizesSource(t *testing.T) {
	sourceID := uuid.New()
	input := CreateCostEntryInput{
		SourceType:  "manual",
		SourceID:    &sourceID,
		Amount:      1.25,
		Description: "",
	}

	if err := prepareAIUsageCostEntryInput(&input); err != nil {
		t.Fatalf("prepareAIUsageCostEntryInput returned error: %v", err)
	}
	if input.SourceType != "ai_usage" {
		t.Fatalf("SourceType = %q, want ai_usage", input.SourceType)
	}
	if input.Description == "" {
		t.Fatalf("Description was not defaulted")
	}
}
