package setup

import (
	"fmt"
	"os"
	"reflect"
	"testing"
)

// helper: build Deps from an installed-set and a set of tools that
// "install" successfully.
func fakeDeps(installed map[string]bool, installOK map[string]bool) Deps {
	return Deps{
		IsInstalled: func(t string) bool { return installed[t] },
		Install: func(t string) error {
			if !installOK[t] {
				return fmt.Errorf("install failed: %s", t)
			}
			installed[t] = true // now present
			return nil
		},
	}
}

func TestRun_AllAbsent(t *testing.T) { // I4
	d := fakeDeps(map[string]bool{}, map[string]bool{"a": true, "b": true})
	r := Run([]string{"a", "b"}, d)
	if !reflect.DeepEqual(r.Installed, []string{"a", "b"}) || len(r.Failed) != 0 {
		t.Fatalf("I4: want installed[a,b], got %+v", r)
	}
}

func TestRun_AllPresent(t *testing.T) { // I5
	d := fakeDeps(map[string]bool{"a": true, "b": true}, nil)
	r := Run([]string{"a", "b"}, d)
	if !reflect.DeepEqual(r.Skipped, []string{"a", "b"}) || len(r.Installed) != 0 {
		t.Fatalf("I5: want skipped[a,b], got %+v", r)
	}
}

func TestRun_Mixed(t *testing.T) { // I6
	d := fakeDeps(map[string]bool{"a": true}, map[string]bool{"b": true})
	r := Run([]string{"a", "b"}, d)
	if !reflect.DeepEqual(r.Skipped, []string{"a"}) || !reflect.DeepEqual(r.Installed, []string{"b"}) {
		t.Fatalf("I6: want skipped[a] installed[b], got %+v", r)
	}
}

func TestRun_InstallFails(t *testing.T) { // I7
	d := fakeDeps(map[string]bool{}, map[string]bool{"a": true}) // b fails
	r := Run([]string{"a", "b"}, d)
	if !reflect.DeepEqual(r.Installed, []string{"a"}) || !reflect.DeepEqual(r.Failed, []string{"b"}) {
		t.Fatalf("I7: want installed[a] failed[b], got %+v", r)
	}
}

func TestRun_InstalledButVerifyFails(t *testing.T) { // I8
	// Install "succeeds" (no error) but tool still not present -> failed.
	d := Deps{
		IsInstalled: func(string) bool { return false },
		Install:     func(string) error { return nil },
	}
	r := Run([]string{"a"}, d)
	if !reflect.DeepEqual(r.Failed, []string{"a"}) {
		t.Fatalf("I8: want failed[a], got %+v", r)
	}
}

func TestRun_MultipleFailures(t *testing.T) { // I14
	d := fakeDeps(map[string]bool{}, map[string]bool{}) // all fail
	r := Run([]string{"a", "b", "c"}, d)
	if !reflect.DeepEqual(r.Failed, []string{"a", "b", "c"}) {
		t.Fatalf("I14: want failed[a,b,c], got %+v", r)
	}
}

func TestRun_Order(t *testing.T) { // E4 deterministic order
	d := fakeDeps(map[string]bool{}, map[string]bool{"x": true, "y": true, "z": true})
	r := Run([]string{"z", "y", "x"}, d)
	if !reflect.DeepEqual(r.Installed, []string{"z", "y", "x"}) {
		t.Fatalf("E4: order must be preserved, got %+v", r.Installed)
	}
}

func TestGoInstallCmd(t *testing.T) { // I13
	got := GoInstallCmd("example.com/x@latest")
	want := []string{"go", "install", "example.com/x@latest"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("I13: got %v", got)
	}
}

func TestHasGo(t *testing.T) { // I3 helper
	if HasGo(func(string) (string, error) { return "", os.ErrNotExist }) {
		t.Fatal("HasGo should be false when go missing")
	}
	if !HasGo(func(string) (string, error) { return "/usr/local/go/bin/go", nil }) {
		t.Fatal("HasGo should be true when go present")
	}
}
