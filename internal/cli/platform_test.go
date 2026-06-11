package cli

import (
	"runtime"
	"testing"
)

func TestParsePlatforms(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    int
		wantErr bool
	}{
		{"empty", nil, 0, false},
		{"all", []string{"all"}, 6, false},
		{"linux", []string{"linux"}, 2, false},
		{"darwin", []string{"darwin"}, 2, false},
		{"windows", []string{"windows"}, 2, false},
		{"single", []string{"linux/amd64"}, 1, false},
		{"multiple", []string{"linux/amd64", "darwin/arm64"}, 2, false},
		{"dedup", []string{"linux/amd64", "linux/amd64"}, 1, false},
		{"os+specific no dup", []string{"linux", "linux/amd64"}, 2, false},
		{"bad os", []string{"freebsd/amd64"}, 0, true},
		{"bad arch", []string{"linux/mips"}, 0, true},
		{"bad format", []string{"linux-amd64"}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePlatforms(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parsePlatforms(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
			if len(got) != tt.want {
				t.Errorf("parsePlatforms(%v) returned %d targets, want %d", tt.args, len(got), tt.want)
			}
		})
	}
}

func TestParsePlatforms_AllContents(t *testing.T) {
	targets, err := parsePlatforms([]string{"all"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"darwin/arm64":  true,
		"darwin/amd64":  true,
		"linux/arm64":   true,
		"linux/amd64":   true,
		"windows/arm64": true,
		"windows/amd64": true,
	}
	for _, tgt := range targets {
		key := tgt.GOOS + "/" + tgt.GOARCH
		if !want[key] {
			t.Errorf("unexpected target %s", key)
		}
		delete(want, key)
	}
	for k := range want {
		t.Errorf("missing target %s", k)
	}
}

func TestPlatformTarget_Label(t *testing.T) {
	tests := []struct {
		target platformTarget
		want   string
	}{
		{platformTarget{"linux", "amd64"}, "linux-amd64"},
		{platformTarget{"darwin", "arm64"}, "darwin-arm64"},
		{platformTarget{"windows", "amd64"}, "windows-amd64"},
	}
	for _, tt := range tests {
		if got := tt.target.label(); got != tt.want {
			t.Errorf("label() = %q, want %q", got, tt.want)
		}
	}
}

func TestPlatformTarget_BinName(t *testing.T) {
	tests := []struct {
		target platformTarget
		base   string
		want   string
	}{
		{platformTarget{"linux", "amd64"}, "myapp", "myapp"},
		{platformTarget{"darwin", "arm64"}, "myapp", "myapp"},
		{platformTarget{"windows", "amd64"}, "myapp", "myapp.exe"},
		{platformTarget{"windows", "amd64"}, "myapp.exe", "myapp.exe"},
	}
	for _, tt := range tests {
		if got := tt.target.binName(tt.base); got != tt.want {
			t.Errorf("binName(%q) for %s = %q, want %q", tt.base, tt.target.GOOS, got, tt.want)
		}
	}
}

func TestPlatformTarget_IsHost(t *testing.T) {
	host := platformTarget{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH}
	if !host.isHost() {
		t.Error("isHost() returned false for runtime GOOS/GOARCH")
	}

	foreign := platformTarget{GOOS: "plan9", GOARCH: "mips"}
	if foreign.isHost() {
		t.Error("isHost() returned true for plan9/mips")
	}
}

func TestResolveOutputPath(t *testing.T) {
	tests := []struct {
		name   string
		base   string
		target platformTarget
		multi  bool
		want   string
	}{
		{
			"single linux",
			"/out/myapp",
			platformTarget{"linux", "amd64"},
			false,
			"/out/myapp",
		},
		{
			"single windows adds exe",
			"/out/myapp",
			platformTarget{"windows", "amd64"},
			false,
			"/out/myapp.exe",
		},
		{
			"single windows already exe",
			"/out/myapp.exe",
			platformTarget{"windows", "amd64"},
			false,
			"/out/myapp.exe",
		},
		{
			"multi linux",
			"/out/myapp",
			platformTarget{"linux", "amd64"},
			true,
			"/out/myapp-linux-amd64",
		},
		{
			"multi windows",
			"/out/myapp",
			platformTarget{"windows", "arm64"},
			true,
			"/out/myapp-windows-arm64.exe",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveOutputPath(tt.base, tt.target, tt.multi)
			if got != tt.want {
				t.Errorf("resolveOutputPath(%q, %s, %v) = %q, want %q",
					tt.base, tt.target.label(), tt.multi, got, tt.want)
			}
		})
	}
}
