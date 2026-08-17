// Package setup installs the JS-finding tools via `go install`.
package setup

// Result records what happened per tool.
type Result struct {
	Installed []string
	Skipped   []string
	Failed    []string
}

// Deps are injected so tests never hit the network.
type Deps struct {
	IsInstalled func(tool string) bool
	Install     func(tool string) error
}

// Run installs each tool that is not already present, then verifies.
// A failure of one tool does not abort the others (all collected).
func Run(tools []string, d Deps) Result {
	var r Result
	for _, t := range tools {
		if d.IsInstalled(t) {
			r.Skipped = append(r.Skipped, t)
			continue
		}
		if err := d.Install(t); err != nil {
			r.Failed = append(r.Failed, t)
			continue
		}
		if d.IsInstalled(t) {
			r.Installed = append(r.Installed, t)
		} else {
			r.Failed = append(r.Failed, t) // installed but verify failed
		}
	}
	return r
}

// GoInstallCmd builds the argv for installing a go package.
func GoInstallCmd(pkg string) []string {
	return []string{"go", "install", pkg}
}

// HasGo reports whether the go toolchain is available.
func HasGo(lookPath func(string) (string, error)) bool {
	_, err := lookPath("go")
	return err == nil
}
