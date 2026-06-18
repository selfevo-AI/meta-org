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

func TestValidateRequirementStatusAllowsConvertedAnalysis(t *testing.T) {
	for _, action := range []string{"analyze", "sync_analysis"} {
		if err := validateRequirementStatus("converted", action); err != nil {
			t.Fatalf("validateRequirementStatus(converted, %s) returned error: %v", action, err)
		}
	}
}

func TestValidateRequirementStatusAllowsProjectConversionFromAnyStatus(t *testing.T) {
	for _, status := range []string{"draft", "analyzed", "approved", "converted"} {
		if err := validateRequirementStatus(status, "convert"); err != nil {
			t.Fatalf("validateRequirementStatus(%s, convert) returned error: %v", status, err)
		}
	}
}
