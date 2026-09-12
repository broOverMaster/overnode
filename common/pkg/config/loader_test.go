package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"overnode/common/pkg/config/schema"
)

type loaderTestConfiguration struct {
	Sample struct {
		Value     string        `mapstructure:"value"`
		Port      uint16        `mapstructure:"port"`
		Retries   int           `mapstructure:"retries"`
		Timeout   time.Duration `mapstructure:"timeout"`
		FilePath  string        `mapstructure:"file_path"`
		Endpoints []string      `mapstructure:"endpoints"`
	} `mapstructure:"sample"`
}

func TestLoadCombinesSourcesWithoutApplicationTypes(t *testing.T) {
	directory := t.TempDir()
	configFile := filepath.Join(directory, "config.yaml")
	contents := "sample:\n  value: from-file\n  port: 1000\n  timeout: 3s\n  file_path: data/example.txt\n"
	if err := os.WriteFile(configFile, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SAMPLE__VALUE", "from-environment")

	var result loaderTestConfiguration
	err := Load(
		[]string{"--config", configFile, "--sample.port", "1080"},
		nil,
		&result,
		loaderTestOptions(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Sample.Value != "from-environment" || result.Sample.Port != 1080 || result.Sample.Timeout != 3*time.Second {
		t.Fatalf("unexpected merged configuration: %+v", result.Sample)
	}
	if result.Sample.FilePath != filepath.Join(directory, "data", "example.txt") {
		t.Fatalf("unexpected normalized path: %q", result.Sample.FilePath)
	}
}

func TestLoadStringSliceSourcesReplaceWholeList(t *testing.T) {
	directory := t.TempDir()
	configFile := filepath.Join(directory, "config.yaml")
	contents := "sample:\n  endpoints:\n    - from-file-one\n    - from-file-two\n"
	if err := os.WriteFile(configFile, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	for name, test := range map[string]struct {
		args []string
		env  string
		want []string
	}{
		"file": {
			args: []string{"--config", configFile},
			want: []string{"from-file-one", "from-file-two"},
		},
		"environment replaces file": {
			args: []string{"--config", configFile},
			env:  "from-env-one,from-env-two",
			want: []string{"from-env-one", "from-env-two"},
		},
		"CLI replaces environment": {
			args: []string{"--config", configFile, "--sample.endpoints", "from-cli-one", "--sample.endpoints", "from-cli-two"},
			env:  "from-env-one,from-env-two",
			want: []string{"from-cli-one", "from-cli-two"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if test.env != "" {
				t.Setenv("SAMPLE__ENDPOINTS", test.env)
			}
			var result loaderTestConfiguration
			if err := Load(test.args, nil, &result, loaderTestOptions()); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(result.Sample.Endpoints, test.want) {
				t.Fatalf("unexpected endpoints: got %v, want %v", result.Sample.Endpoints, test.want)
			}
		})
	}
}

func TestLoadIntFromAllSources(t *testing.T) {
	directory := t.TempDir()
	configFile := filepath.Join(directory, "config.yaml")
	if err := os.WriteFile(configFile, []byte("sample:\n  retries: 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	for name, test := range map[string]struct {
		args []string
		env  string
		want int
	}{
		"default":     {want: 1},
		"file":        {args: []string{"--config", configFile}, want: 2},
		"environment": {args: []string{"--config", configFile}, env: "3", want: 3},
		"CLI":         {args: []string{"--config", configFile, "--sample.retries", "4"}, env: "3", want: 4},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("SAMPLE__RETRIES", test.env)
			var result loaderTestConfiguration
			if err := Load(test.args, nil, &result, loaderTestOptions()); err != nil {
				t.Fatal(err)
			}
			if result.Sample.Retries != test.want {
				t.Fatalf("got %d, want %d", result.Sample.Retries, test.want)
			}
		})
	}
}

func TestLoadPrintsConfiguredHelp(t *testing.T) {
	var output bytes.Buffer
	var result loaderTestConfiguration
	err := Load([]string{"--help"}, &output, &result, loaderTestOptions())
	if !errors.Is(err, ErrHelp) {
		t.Fatalf("expected ErrHelp, got %v", err)
	}
	for _, expected := range []string{"Usage: example [flags]", "Example configuration loader.", "--sample.timeout duration"} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("help does not contain %q:\n%s", expected, output.String())
		}
	}
}

func TestLoadRejectsInvalidBoundaryArguments(t *testing.T) {
	valid := loaderTestOptions()
	for name, test := range map[string]struct {
		destination any
		options     Options
	}{
		"nil destination":   {nil, valid},
		"value destination": {loaderTestConfiguration{}, valid},
		"empty command":     {&loaderTestConfiguration{}, Options{Fields: valid.Fields}},
	} {
		t.Run(name, func(t *testing.T) {
			if err := Load(nil, nil, test.destination, test.options); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func loaderTestOptions() Options {
	return Options{
		Command:     "example",
		Description: "Example configuration loader.",
		Fields: []schema.Field{
			schema.String("sample.value", "default", "sample value"),
			schema.Uint16("sample.port", 80, "sample port"),
			schema.Int("sample.retries", 1, "sample retries"),
			schema.Duration("sample.timeout", time.Second, "sample timeout"),
			schema.String("sample.file_path", "", "sample file path"),
			schema.StringSlice("sample.endpoints", nil, "sample endpoints"),
		},
	}
}
