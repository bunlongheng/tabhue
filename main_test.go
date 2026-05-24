package main

import "testing"

func TestHexToRGB(t *testing.T) {
	cases := []struct {
		in      string
		r, g, b int
		wantErr bool
	}{
		{"#2563EB", 0x25, 0x63, 0xEB, false},
		{"2563EB", 0x25, 0x63, 0xEB, false},
		{"  #FF3B30 ", 0xFF, 0x3B, 0x30, false},
		{"#fff", 0, 0, 0, true},   // wrong length
		{"#zzzzzz", 0, 0, 0, true}, // not hex
	}
	for _, c := range cases {
		r, g, b, err := hexToRGB(c.in)
		if (err != nil) != c.wantErr {
			t.Errorf("hexToRGB(%q) err=%v, wantErr=%v", c.in, err, c.wantErr)
			continue
		}
		if !c.wantErr && (r != c.r || g != c.g || b != c.b) {
			t.Errorf("hexToRGB(%q) = %d,%d,%d; want %d,%d,%d", c.in, r, g, b, c.r, c.g, c.b)
		}
	}
}

func TestFindProject(t *testing.T) {
	cfg := &Config{Projects: []Project{
		{Path: "/a/b", Color: "#111111"},
		{Path: "/a/b/c", Color: "#222222"},
		{Path: "/x", Color: "#333333"},
	}}
	cases := []struct {
		dir, want string // want = matched path, "" means no match
	}{
		{"/a/b", "/a/b"},
		{"/a/b/c", "/a/b/c"},   // exact, deepest
		{"/a/b/c/d", "/a/b/c"}, // child of deepest match
		{"/a/b/z", "/a/b"},     // child of shallow match
		{"/x", "/x"},
		{"/nope", ""},
		{"/", ""},
	}
	for _, c := range cases {
		p, ok := cfg.findProject(c.dir)
		got := ""
		if ok {
			got = p.Path
		}
		if got != c.want {
			t.Errorf("findProject(%q) = %q; want %q", c.dir, got, c.want)
		}
	}
}

func TestTitle(t *testing.T) {
	cases := []struct {
		p    Project
		want string
	}{
		{Project{Path: "/a/myapp"}, "MYAPP"},                    // default: folder name uppercased
		{Project{Path: "/a/myapp", Label: "Custom"}, "Custom"},  // explicit label
		{Project{Path: "/a/x", Label: "API", Icon: "🔥"}, "🔥 API"}, // icon + label
	}
	for _, c := range cases {
		if got := c.p.title(); got != c.want {
			t.Errorf("title(%+v) = %q; want %q", c.p, got, c.want)
		}
	}
}
