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
	Silent       bool
}

// Parse builds user configuration from CLI args.
func Parse(version string, args []string) (Config, error) {
	fs := tinyflags.NewFlagSet("kustomizer", tinyflags.ContinueOnError)
	fs.Version(version)
	fs.RequirePositional(1)

	cfg := Config{}

	fs.StringSliceVar(&cfg.SkipPatterns, "skip", []string{}, "Skip resources (comma-separated).").
		Short("s").
		Value()
	fs.BoolVar(&cfg.NoGitIgnore, "no-gitignore", false, "Disable .gitignore processing.").
		Short("g").
		Value()
	fs.BoolVar(&cfg.IncludeDot, "include-dot", false, "Include hidden files and directories.").
		Short("i").
		Value()

	fs.BoolVar(&cfg.NoDirSlash, "no-dir-slash", false, "Disable trailing slash for directory resources.").
		Short("D").
		Value()

	fs.BoolVar(&cfg.NoDirFirst, "no-dir-first", false, "Disable directory-first sorting.").
		Short("F").
		Value()

	fs.CounterVar(&cfg.Verbosity, "verbose", 0, "Increase verbosity.").
		Short("v").
		OneOfGroup("logging").
		Value()
	fs.BoolVar(&cfg.Silent, "silent", false, "Suppress per-kustomization no-op logs.").
		Short("q").
		OneOfGroup("logging").
		Value()

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	cfg.BaseDirs = fs.Args()

	return cfg, nil
}
