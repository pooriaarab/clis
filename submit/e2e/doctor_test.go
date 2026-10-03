package e2e

import "testing"

func TestDoctorHealthy(t *testing.T) {
	sb := newSandbox(t)
	sb.Env["SUBMIT_SITES"] = sb.writeSites()
	sb.Run("track", "init")

	r := sb.Run("doctor")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "sites: configured (2 sites, source: file")
	wantContains(t, "stdout", r.Stdout, "store: ok")
	wantContains(t, "stdout", r.Stdout, "directories: 32 built-in (ok)")

	r = sb.Run("doctor", "--json")
	wantExit(t, r, 0)
	v := r.JSON(t)
	if v["ok"] != true {
		t.Errorf("json ok = %v", v["ok"])
	}
}

func TestDoctorMissingSites(t *testing.T) {
	sb := newSandbox(t)

	r := sb.Run("doctor")
	wantExit(t, r, 0)
	wantContains(t, "stdout", r.Stdout, "sites: not configured")

	r = sb.Run("doctor", "--json")
	v := r.JSON(t)
	if v["ok"] != false {
		t.Errorf("json ok = %v, want false", v["ok"])
	}
}
