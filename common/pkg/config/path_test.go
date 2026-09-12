package config

import (
	"path/filepath"
	"testing"
)

type pathTestConfig struct {
	Logging struct {
		FilePath string `mapstructure:"file_path"`
		Label    string `mapstructure:"label"`
	} `mapstructure:"logging"`
	Optional struct {
		CachePath string `mapstructure:"cache_path"`
	} `mapstructure:"optional"`
}

func TestResolvePathFieldsRelativeToConfigFile(t *testing.T) {
	directory := t.TempDir()
	configuration := pathTestConfig{}
	configuration.Logging.FilePath = "logs/../overnode.log"
	configuration.Logging.Label = "relative-value"

	if err := resolvePathFields(&configuration, filepath.Join(directory, "config", "node.yaml")); err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(directory, "config", "overnode.log")
	if configuration.Logging.FilePath != expected {
		t.Fatalf("unexpected file path: %q", configuration.Logging.FilePath)
	}
	if configuration.Logging.Label != "relative-value" {
		t.Fatalf("non-path field was changed: %q", configuration.Logging.Label)
	}
}

func TestResolvePathFieldsRelativeToWorkingDirectory(t *testing.T) {
	configuration := pathTestConfig{}
	configuration.Logging.FilePath = "logs/overnode.log"

	if err := resolvePathFields(&configuration, ""); err != nil {
		t.Fatal(err)
	}
	expected, err := filepath.Abs(filepath.Join("logs", "overnode.log"))
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Logging.FilePath != expected {
		t.Fatalf("unexpected file path: %q", configuration.Logging.FilePath)
	}
}

func TestResolvePathFieldsCleansAbsolutePathAndPreservesEmptyPath(t *testing.T) {
	directory := t.TempDir()
	configuration := pathTestConfig{}
	configuration.Logging.FilePath = filepath.Join(directory, "logs", "..", "overnode.log")

	if err := resolvePathFields(&configuration, ""); err != nil {
		t.Fatal(err)
	}
	if configuration.Logging.FilePath != filepath.Join(directory, "overnode.log") {
		t.Fatalf("unexpected file path: %q", configuration.Logging.FilePath)
	}
	if configuration.Optional.CachePath != "" {
		t.Fatalf("empty path was changed: %q", configuration.Optional.CachePath)
	}
}
