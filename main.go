// Command tabhue colors your terminal tab by the project directory you're in.
//
// It matches your current directory against a small JSON config and prints the
// terminal escape codes (OSC sequences) that iTerm2 (and compatible terminals)
// use to set the tab color and title - so every project gets an instant,
// recognizable tab. Wire `tabhue apply` into a shell hook (precmd/chpwd) and
// your tabs recolor as you move between projects.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const version = "0.1.0"

// OSC (Operating System Command) sequences start with ESC and end with BEL.
const (
	esc = "\033"
	bel = "\a"
)

// Project is one entry in the config: a directory and how its tab should look.
type Project struct {
	Path  string `json:"path"`            // absolute project dir; matched against $PWD and its parents
	Label string `json:"label,omitempty"` // tab title (defaults to the folder name, uppercased)
	Color string `json:"color"`           // hex like "#2563EB"
	Icon  string `json:"icon,omitempty"`  // optional emoji shown before the label
}

// Config is the whole file: just a list of projects.
type Config struct {
	Projects []Project `json:"projects"`
}

// configPath returns the config location, overridable via $TABHUE_CONFIG.
func configPath() string {
	if v := os.Getenv("TABHUE_CONFIG"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "tabhue", "config.json")
}

func loadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", configPath(), err)
	}
	return &c, nil
}

// hexToRGB turns "#2563EB" (or "2563EB") into its three 0-255 channels.
func hexToRGB(hex string) (r, g, b int, err error) {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(hex) != 6 {
		return 0, 0, 0, fmt.Errorf("bad hex color %q (want #RRGGBB)", hex)
	}
	v, err := strconv.ParseInt(hex, 16, 32)
	if err != nil {
		return 0, 0, 0, err
	}
	return int(v>>16) & 0xff, int(v>>8) & 0xff, int(v) & 0xff, nil
}

// findProject walks up from dir, returning the first project whose Path is dir
// or an ancestor of it. The deepest match wins so nested projects work.
func (c *Config) findProject(dir string) (*Project, bool) {
	dir = filepath.Clean(dir)
	var best *Project
	bestLen := -1
	for i := range c.Projects {
		p := filepath.Clean(c.Projects[i].Path)
		if dir == p || strings.HasPrefix(dir, p+string(os.PathSeparator)) {
			if len(p) > bestLen {
				best, bestLen = &c.Projects[i], len(p)
			}
		}
	}
	return best, best != nil
}

// title is the tab text: "<icon> <LABEL>", defaulting to the folder name.
func (p *Project) title() string {
	label := p.Label
	if label == "" {
		label = strings.ToUpper(filepath.Base(p.Path))
	}
	if p.Icon != "" {
		return p.Icon + " " + label
	}
	return label
}

// emitTab prints the OSC codes that color the current tab and set its title.
// With bg, it also tints the terminal background to a 35% blend of the color
// over a near-black base (kept subtle and readable).
func emitTab(p *Project, bg bool) error {
	r, g, b, err := hexToRGB(p.Color)
	if err != nil {
		return err
	}
	// OSC 6 - iTerm2 tab color, one sequence per channel.
	fmt.Printf("%s]6;1;bg;red;brightness;%d%s", esc, r, bel)
	fmt.Printf("%s]6;1;bg;green;brightness;%d%s", esc, g, bel)
	fmt.Printf("%s]6;1;bg;blue;brightness;%d%s", esc, b, bel)
	if bg {
		blend := func(c int) int { return 10 + c*35/100 }
		fmt.Printf("%s]11;rgb:%02x/%02x/%02x%s", esc, blend(r), blend(g), blend(b), bel)
	}
	// OSC 1 - tab title; OSC 2 - window title (kept in sync).
	t := p.title()
	fmt.Printf("%s]1;%s%s", esc, t, bel)
	fmt.Printf("%s]2;%s%s", esc, t, bel)
	return nil
}

// resetTab clears the tab color and background back to the profile defaults.
func resetTab() {
	fmt.Printf("%s]6;1;bg;*;default%s", esc, bel) // reset tab color
	fmt.Printf("%s]111%s", esc, bel)              // reset background color
}

func cmdApply(args []string) {
	bg, dir := false, ""
	for _, a := range args {
		switch a {
		case "--bg", "-bg":
			bg = true
		default:
			if !strings.HasPrefix(a, "-") {
				dir = a
			}
		}
	}
	if dir == "" {
		dir, _ = os.Getwd()
	}
	cfg, err := loadConfig()
	if err != nil {
		fail(err)
	}
	p, ok := cfg.findProject(dir)
	if !ok {
		return // no match: silent no-op, safe to wire into a shell hook
	}
	if err := emitTab(p, bg); err != nil {
		fail(err)
	}
}

func cmdList() {
	cfg, err := loadConfig()
	if err != nil {
		fail(err)
	}
	if len(cfg.Projects) == 0 {
		fmt.Println("no projects configured -", configPath())
		return
	}
	for _, p := range cfg.Projects {
		fmt.Printf("%-9s %-22s %s\n", p.Color, p.title(), p.Path)
	}
}

func cmdInit() {
	path := configPath()
	if _, err := os.Stat(path); err == nil {
		fmt.Println("config already exists:", path)
		return
	}
	home, _ := os.UserHomeDir()
	sample := Config{Projects: []Project{
		{Path: filepath.Join(home, "Sites", "myapp"), Label: "MYAPP", Color: "#2563EB", Icon: "\U0001F4CE"},
	}}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		fail(err)
	}
	data, _ := json.MarshalIndent(sample, "", "  ")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		fail(err)
	}
	fmt.Println("wrote sample config:", path)
}

func usage() {
	fmt.Print(`tabhue - color your terminal tab by project directory

usage:
  tabhue [apply] [dir] [--bg]   color the tab for dir (default: current dir)
  tabhue list                   list configured projects
  tabhue init                   write a sample config
  tabhue reset                  clear the tab color/background
  tabhue version

config: ~/.config/tabhue/config.json  (override with $TABHUE_CONFIG)
`)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "tabhue:", err)
	os.Exit(1)
}

func main() {
	args := os.Args[1:]
	cmd := "apply"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "apply":
		cmdApply(args)
	case "list":
		cmdList()
	case "init":
		cmdInit()
	case "reset":
		resetTab()
	case "version", "-v", "--version":
		fmt.Println("tabhue", version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "tabhue: unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
}
