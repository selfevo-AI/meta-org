package metaorg

import (
	"testing"
	"time"
)

func TestFilterInboxItemsByType(t *testing.T) {
	items := []InboxItem{
		{ID: "tool-1", Type: "tool_approval"},
		{ID: "finance-1", Type: "finance_export"},
	}

	filtered := filterInboxItems(items, "finance_export")

	if len(filtered) != 1 {
		t.Fatalf("filtered length = %d, want 1", len(filtered))
	}
	if filtered[0].ID != "finance-1" {
		t.Fatalf("filtered ID = %q, want finance-1", filtered[0].ID)
	}
}

func TestSortInboxItemsOrdersByPriorityThenNewest(t *testing.T) {
	now := time.Unix(100, 0).UTC()
	items := []InboxItem{
		{ID: "low-new", Priority: "low", CreatedAt: now.Add(3 * time.Minute)},
		{ID: "high-old", Priority: "high", CreatedAt: now},
		{ID: "critical", Priority: "critical", CreatedAt: now.Add(time.Minute)},
		{ID: "high-new", Priority: "high", CreatedAt: now.Add(2 * time.Minute)},
	}

	sortInboxItems(items)

	got := []string{items[0].ID, items[1].ID, items[2].ID, items[3].ID}
	want := []string{"critical", "high-new", "high-old", "low-new"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q; got order %v", i, got[i], want[i], got)
		}
	}
}
