package schema

import "testing"

func TestCombineRejectsAmbiguousKeys(t *testing.T) {
	for _, keys := range [][]string{{"a", "a.b"}, {"a.b", "a"}, {"a.b", "a__b"}, {"A.b"}, {"a..b"}, {"-bad"}, {"a b"}} {
		var fields []Field
		for _, key := range keys {
			fields = append(fields, String(key, "", "value"))
		}
		if _, err := Combine(fields); err == nil {
			t.Fatalf("expected error for %v", keys)
		}
	}
}
