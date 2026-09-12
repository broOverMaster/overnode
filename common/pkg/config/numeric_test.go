package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRejectsOutOfRangePortsFromFilesAndEnvironment(t *testing.T) {
	for _, raw := range []string{"-1", "65536", "1.5", "18446744073709551616"} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("SAMPLE__PORT", raw)
			var got loaderTestConfiguration
			if err := Load(nil, nil, &got, loaderTestOptions()); err == nil {
				t.Fatal("environment accepted invalid port")
			}
			t.Setenv("SAMPLE__PORT", "")
			for ext, body := range map[string]string{
				"json": `{"sample":{"port":` + raw + `}}`,
				"yaml": "sample:\n  port: " + raw + "\n",
				"toml": "[sample]\nport = " + raw + "\n",
			} {
				file := filepath.Join(t.TempDir(), "invalid."+ext)
				if err := os.WriteFile(file, []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
				if err := Load([]string{"--config", file}, nil, &got, loaderTestOptions()); err == nil {
					t.Errorf("%s accepted invalid port", ext)
				}
			}
		})
	}
}

func TestLoadAcceptsUint16Boundaries(t *testing.T) {
	for _, raw := range []string{"0", "65535"} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("SAMPLE__PORT", raw)
			var got loaderTestConfiguration
			if err := Load(nil, nil, &got, loaderTestOptions()); err != nil {
				t.Fatal(err)
			}
			if raw == "0" && got.Sample.Port != 0 {
				t.Fatal(got.Sample.Port)
			}
			if raw == "65535" && got.Sample.Port != 65535 {
				t.Fatal(got.Sample.Port)
			}
		})
	}
}
