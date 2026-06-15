package metaresource

import "testing"

func TestListFromJSONAcceptsArraysAndSingleObject(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want int
	}{
		{name: "string array", data: []byte(`["accepted","verified"]`), want: 2},
		{name: "object array", data: []byte(`[{"name":"capability"}]`), want: 1},
		{name: "single object", data: []byte(`{"name":"capability"}`), want: 1},
		{name: "invalid", data: []byte(`"capability"`), want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := listFromJSON(tt.data)
			if len(got) != tt.want {
				t.Fatalf("len(listFromJSON(%s)) = %d, want %d", tt.data, len(got), tt.want)
			}
		})
	}
}

func TestMustJSONPreservesEmptyLists(t *testing.T) {
	if got := string(mustJSON([]any(nil))); got != "[]" {
		t.Fatalf("mustJSON([]any(nil)) = %s, want []", got)
	}
	if got := string(mustJSON([]map[string]any(nil))); got != "[]" {
		t.Fatalf("mustJSON([]map[string]any(nil)) = %s, want []", got)
	}
}
