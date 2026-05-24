package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStdout runs f and returns everything it printed to os.Stdout.
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestEmitTab(t *testing.T) {
	p := &Project{Path: "/x/api", Label: "API", Color: "#2563EB"}
	out := captureStdout(func() {
		if err := emitTab(p, false); err != nil {
			t.Fatal(err)
		}
	})
	for _, want := range []string{
		"\033]6;1;bg;red;brightness;37\a",
		"\033]6;1;bg;green;brightness;99\a",
		"\033]6;1;bg;blue;brightness;235\a",
		"\033]1;API\a",
		"\033]2;API\a",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("emitTab output missing %q", want)
		}
	}
	if strings.Contains(out, "]11;") {
		t.Error("emitTab without --bg should not set a background")
	}
}

func TestEmitTabBg(t *testing.T) {
	// 35% blend over base 10: r=37->22=0x16, g=99->44=0x2c, b=235->92=0x5c
	out := captureStdout(func() { emitTab(&Project{Path: "/x", Label: "X", Color: "#2563EB"}, true) })
	if !strings.Contains(out, "\033]11;rgb:16/2c/5c\a") {
		t.Errorf("emitTab --bg missing expected OSC 11 blend; got %q", out)
	}
}

func TestEmitTabBadColor(t *testing.T) {
	if err := emitTab(&Project{Path: "/x", Color: "nope"}, false); err == nil {
		t.Error("expected error for bad color")
	}
}

func TestResetTab(t *testing.T) {
	out := captureStdout(resetTab)
	if !strings.Contains(out, "]6;1;bg;*;default") || !strings.Contains(out, "]111") {
		t.Errorf("resetTab output unexpected: %q", out)
	}
}

func TestConfigPathOverride(t *testing.T) {
	t.Setenv("TABHUE_CONFIG", "/tmp/custom.json")
	if got := configPath(); got != "/tmp/custom.json" {
		t.Errorf("configPath() = %q; want /tmp/custom.json", got)
	}
}

func TestLoadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte(`{"projects":[{"path":"/a","label":"A","color":"#111111"}]}`), 0o644)
	t.Setenv("TABHUE_CONFIG", path)
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Projects) != 1 || cfg.Projects[0].Label != "A" {
		t.Errorf("loadConfig parsed wrong: %+v", cfg.Projects)
	}
}

func TestCmdInit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tw", "config.json")
	t.Setenv("TABHUE_CONFIG", path)
	captureStdout(cmdInit)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cmdInit did not write config: %v", err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		t.Errorf("cmdInit wrote invalid JSON: %v", err)
	}
	if len(c.Projects) == 0 {
		t.Error("cmdInit sample has no projects")
	}
	if out := captureStdout(cmdInit); !strings.Contains(out, "already exists") {
		t.Errorf("second cmdInit should report already exists; got %q", out)
	}
}

func TestCmdApplyAndList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte(`{"projects":[{"path":"/tmp","label":"TMP","color":"#34C759","icon":"x"}]}`), 0o644)
	t.Setenv("TABHUE_CONFIG", path)
	if out := captureStdout(func() { cmdApply([]string{"/tmp"}) }); !strings.Contains(out, "]1;x TMP\a") {
		t.Errorf("cmdApply missing title; got %q", out)
	}
	if out := captureStdout(func() { cmdApply([]string{"/no/match"}) }); out != "" {
		t.Errorf("cmdApply on no-match should be silent; got %q", out)
	}
	if out := captureStdout(cmdList); !strings.Contains(out, "TMP") || !strings.Contains(out, "/tmp") {
		t.Errorf("cmdList missing entry; got %q", out)
	}
}
