package cli

import (
	"reflect"
	"testing"
)

func TestFirestoreValueRoundTrip(t *testing.T) {
	input := map[string]any{
		"title":   "Hero",
		"ordinal": 3,
		"enabled": true,
		"missing": nil,
		"items": []any{
			"alpha",
			7,
			false,
			nil,
			map[string]any{
				"nested": "value",
			},
		},
	}

	got := fsDecode(fsEncode(input))
	if !reflect.DeepEqual(got, input) {
		t.Fatalf("fsDecode(fsEncode(input)) = %#v, want %#v", got, input)
	}
}
