package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"overnode/common/pkg/config/schema"
)

func TestLoadRejectsReservedKeys(t *testing.T) {
	for _, key := range []string{"config", "help"} {
		var result struct{}
		if err := Load(nil, nil, &result, Options{Command: "test", Fields: []schema.Field{schema.String(key, "", "value")}}); err == nil {
			t.Fatalf("expected reserved key error for %q", key)
		}
	}
}

func TestLoadRejectsNonStructDestinations(t *testing.T) {
	var pointer *loaderTestConfiguration
	for _, destination := range []any{pointer, new(int), new(map[string]any), &pointer} {
		if err := Load(nil, nil, destination, loaderTestOptions()); err == nil {
			t.Fatalf("expected destination error for %T", destination)
		}
	}
}

func TestLoadFormatsAndPriority(t *testing.T) {
	t.Setenv("SAMPLE__VALUE", "")
	for extension, contents := range map[string]string{
		"yaml": "sample:\n  value: file\n",
		"toml": "[sample]\nvalue = 'file'\n",
		"json": `{"sample":{"value":"file"}}`,
	} {
		t.Run(extension, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config."+extension)
			if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
				t.Fatal(err)
			}
			var result loaderTestConfiguration
			load := func(args []string, want string) {
				t.Helper()
				if err := Load(args, nil, &result, loaderTestOptions()); err != nil {
					t.Fatal(err)
				}
				if result.Sample.Value != want {
					t.Fatalf("got %q, want %q", result.Sample.Value, want)
				}
			}
			load(nil, "default")
			load([]string{"--config", path}, "file")
			t.Setenv("SAMPLE__VALUE", "environment")
			load([]string{"--config", path}, "environment")
			load([]string{"--config", path, "--sample.value", "cli"}, "cli")
		})
	}
}

func TestLoadInputErrorsAndHelp(t *testing.T) {
	var result loaderTestConfiguration
	for _, args := range [][]string{{"--unknown"}, {"unexpected"}, {"--sample.port", "65536"}, {"--config", filepath.Join(t.TempDir(), "missing.yaml")}} {
		if err := Load(args, nil, &result, loaderTestOptions()); err == nil {
			t.Fatalf("expected error for %v", args)
		}
	}
	path := filepath.Join(t.TempDir(), "unknown.yaml")
	if err := os.WriteFile(path, []byte("unknown: true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Load([]string{"--config", path}, nil, &result, loaderTestOptions()); err == nil {
		t.Fatal("expected unknown key error")
	}
	if err := Load([]string{"-h", "--config", "missing.yaml"}, nil, &result, loaderTestOptions()); !errors.Is(err, ErrHelp) {
		t.Fatalf("expected help, got %v", err)
	}
}

func TestResolvePathsSkipsCyclesAndPrivateFields(t *testing.T) {
	type node struct {
		FilePath string `mapstructure:"file_path"`
		Next     *node  `mapstructure:"next"`
		private  *node
	}
	value := node{FilePath: "data.txt", private: &node{FilePath: "private.txt"}}
	value.Next = &value
	if err := resolvePathFields(&value, ""); err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(value.FilePath) || value.private.FilePath != "private.txt" {
		t.Fatalf("unexpected paths: %+v", value)
	}
}
