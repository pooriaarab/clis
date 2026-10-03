package e2e

import "testing"

func TestVersion(t *testing.T) {
	sb := newSandbox(t)

	r := sb.Run("version")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "submit 0.1.0")

	r = sb.Run("version", "--json")
	wantExit(t, r, 0)
	if v := r.JSON(t); v["version"] != "0.1.0" {
		t.Errorf("version = %v", v)
	}
}

func TestHelpAndUnknownCommand(t *testing.T) {
	sb := newSandbox(t)

	r := sb.Run("--help")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "version")

	r = sb.Run("nosuch")
	wantExit(t, r, 2)
}

func TestErrorOutputFormat(t *testing.T) {
	sb := newSandbox(t)

	r := sb.Run("nosuch", "--json")
	wantExit(t, r, 2)
	if v := r.JSON(t); v["ok"] != false {
		t.Errorf("json ok = %v, want false", v["ok"])
	}

	r = sb.Run("nosuch", "--json=false")
	wantExit(t, r, 2)
	wantContains(t, "stderr", r.Stderr, "error:")
	if r.Stdout != "" {
		t.Errorf("stdout = %q, want empty in text mode", r.Stdout)
	}
}
