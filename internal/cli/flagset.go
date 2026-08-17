package cli

import (
	"flag"
	"io"
	"time"
)

// flagSet is a thin wrapper over flag.FlagSet with usage strings dropped
// (help text lives in banner.go, not per-flag).
type flagSet struct{ set *flag.FlagSet }

func newFlagSet(out io.Writer) *flagSet {
	fs := flag.NewFlagSet("fetch", flag.ContinueOnError)
	fs.SetOutput(out)
	return &flagSet{set: fs}
}

func (f *flagSet) String(name, def string) *string { return f.set.String(name, def, "") }
func (f *flagSet) Bool(name string) *bool           { return f.set.Bool(name, false, "") }
func (f *flagSet) Int(name string, def int) *int    { return f.set.Int(name, def, "") }
func (f *flagSet) Duration(name string, def time.Duration) *time.Duration {
	return f.set.Duration(name, def, "")
}
func (f *flagSet) Parse(args []string) error { return f.set.Parse(args) }
