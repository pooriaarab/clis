package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCacheDirResolution(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	xdg := filepath.Join(t.TempDir(), "xdg")

	cases := []struct {
		name, flag, merx, xdg, want string
	}{
		{"flag wins", "/from-flag", "/from-env", xdg, "/from-flag"},
		{"MERX_CACHE_DIR", "", "/from-env", xdg, "/from-env"},
		{"XDG_CACHE_HOME", "", "", xdg, filepath.Join(xdg, "merx")},
		{"home cache", "", "", "", filepath.Join(home, ".cache", "merx")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			flagCache = tc.flag
			t.Cleanup(func() { flagCache = "" })
			t.Setenv("MERX_CACHE_DIR", tc.merx)
			t.Setenv("XDG_CACHE_HOME", tc.xdg)
			got, err := cacheDir()
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestDiskUsedCountsNestedFiles(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	payload := []byte("hello merx")
	if err := os.WriteFile(filepath.Join(nested, "c.txt"), payload, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := diskUsed(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := int64(len(payload))
	if got != want {
		t.Fatalf("got %d want %d", got, want)
	}
}
