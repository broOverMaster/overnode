package schema

import (
	"strings"
	"testing"
	"time"
)

func TestCombinePreservesTypedFields(t *testing.T) {
	fields, err := Combine(
		[]Field{String("component.name", "default", "component name")},
		[]Field{
			Uint16("component.port", 1080, "component port"),
			Int("component.retries", 3, "component retries"),
			Duration("component.timeout", 2*time.Second, "component timeout"),
			StringSlice("component.endpoints", []string{"one"}, "component endpoints"),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 5 || fields[0].Kind != KindString || fields[1].Kind != KindUint16 || fields[2].Kind != KindInt || fields[3].Kind != KindDuration || fields[4].Kind != KindStringSlice {
		t.Fatalf("unexpected combined fields: %+v", fields)
	}
}

func TestCombineRejectsInvalidSchemas(t *testing.T) {
	for name, groups := range map[string][][]Field{
		"empty key":        {{String("", "", "usage")}},
		"empty usage":      {{String("component.name", "", "")}},
		"duplicate":        {{String("component.name", "", "usage")}, {String("component.name", "", "usage")}},
		"unsupported kind": {{{Key: "component.name", Kind: 255, Default: "", Usage: "usage"}}},
		"wrong default":    {{{Key: "component.port", Kind: KindUint16, Default: "1080", Usage: "usage"}}},
		"wrong list":       {{{Key: "component.list", Kind: KindStringSlice, Default: "one", Usage: "usage"}}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Combine(groups...); err == nil || strings.TrimSpace(err.Error()) == "" {
				t.Fatalf("expected a descriptive schema error, got %v", err)
			}
		})
	}
}

func TestStringSliceCopiesDefault(t *testing.T) {
	input := []string{"one"}
	field := StringSlice("component.endpoints", input, "component endpoints")
	input[0] = "changed"
	if got := field.Default.([]string)[0]; got != "one" {
		t.Fatalf("constructor retained caller slice: %q", got)
	}
}
