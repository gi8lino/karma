package cli

import (
	"github.com/containeroo/tinyflags"
)

// Config holds parsed command-line options.
type Config struct {
	BaseDirs     []string
	SkipPatterns []string
	Verbosity    int
	NoDirSlash   bool
	NoDirFirst   bool
	NoGitIgnore  bool
	IncludeDot   bool
	Mute         bool
}

// Parse builds user configuration from CLI args.
func Parse(version string, args []string) (Config, error) {
	fs := tinyflags.NewFlagSet("karma", tinyflags.ContinueOnError)
	fs.Version(version)
	fs.RequirePositional(1)
	fs.Note("\n*) skip supports `*` wildcards, `/*` to skip a directory's kustomization without entering it, and `/**` to skip the kustomization but still descend into its children (so those nested dirs can still be handled separately).")

	cfg := Config{}

	// selection
	fs.StringSliceVar(&cfg.SkipPatterns, "skip", []string{}, "Skip resources (comma-separated). *").
		Short("s").
		Value()
	fs.BoolVar(&cfg.NoGitIgnore, "no-gitignore", false, "Disable .gitignore processing.").
		Short("g").
		Value()
	fs.BoolVar(&cfg.IncludeDot, "include-dot", false, "Include hidden files and directories.").
		Short("i").
		Value()

	// formatting
	fs.BoolVar(&cfg.NoDirSlash, "no-dir-slash", false, "Disable trailing slash for directory resources.").
		Short("D").
		Value()
	fs.BoolVar(&cfg.NoDirFirst, "no-dir-first", false, "Disable directory-first sorting.").
		Short("F").
		Value()

	// logging
	fs.CounterVar(&cfg.Verbosity, "verbose", 0, "Increase verbosity.").
		Short("v").
		OneOfGroup("logging").
		Value()
	fs.BoolVar(&cfg.Mute, "mute", false, "Suppress all output.").
		Finalize(func(v bool) bool {
			if v {
				cfg.Verbosity = -1 // LevelOff
			}
			return v
		}).
		Short("q").
		OneOfGroup("logging").
		Value()

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	cfg.BaseDirs = fs.Args()

	return cfg, nil
}
