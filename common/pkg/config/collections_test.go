package config

import (
	"path/filepath"
	"testing"
)

func TestResolvePathsInSiteCollections(t *testing.T) {
	type site struct {
		RootPath string `mapstructure:"root_path"`
	}
	value := struct {
		List     []site
		ByName   map[string]site
		Pointers map[string]*site
		Any      any
	}{
		List:     []site{{RootPath: "list"}},
		ByName:   map[string]site{"alice.overspace": {RootPath: "map"}},
		Pointers: map[string]*site{"bob.overspace": {RootPath: "pointer"}},
		Any:      site{RootPath: "interface"},
	}
	base := t.TempDir()
	if err := resolvePathFields(&value, filepath.Join(base, "node.toml")); err != nil {
		t.Fatal(err)
	}
	for name, actual := range map[string]string{
		"list":      value.List[0].RootPath,
		"map":       value.ByName["alice.overspace"].RootPath,
		"pointer":   value.Pointers["bob.overspace"].RootPath,
		"interface": value.Any.(site).RootPath,
	} {
		if actual != filepath.Join(base, name) {
			t.Errorf("%s: %q", name, actual)
		}
	}
}

func TestResolvePathsRejectsCollectionCycle(t *testing.T) {
	loop := map[string]any{}
	loop["self"] = loop
	value := struct{ Data map[string]any }{loop}
	if err := resolvePathFields(&value, ""); err == nil {
		t.Fatal("expected bounded nesting error")
	}
}
