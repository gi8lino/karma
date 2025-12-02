package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/containeroo/tinyflags"
	"github.com/gi8lino/karma/internal/processor"
)

// Config holds parsed command-line options.
type Config struct {
	BaseDirs      []string
	SkipPatterns  []string
	Verbosity     int
	NoDirSlash    bool
	NoGitIgnore   bool
	IncludeDot    bool
	Mute          bool
	ResourceOrder []string
}

// Parse builds user configuration from CLI args.
func Parse(version string, args []string) (Config, error) {
	fs := tinyflags.NewFlagSet("karma", tinyflags.ContinueOnError)
	fs.Version(version)
	fs.RequirePositional(1)
	fs.Note("*) skip accepts `*` wildcards plus `/*` to ignore a directory's contents and `/**` to ignore the directory while still descending into its children.")

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
	allowed := strings.Join(processor.DefaultResourceOrder(), ", ")
	order := fs.String("order", allowed, fmt.Sprintf("Order resource groups. Allowed: %s.", allowed)).
		Validate(func(v string) error {
			dro := processor.DefaultResourceOrder()
			for _, item := range strings.Split(v, ",") {
				if item == "" {
					continue
				}
				if !slices.Contains(dro, item) {
					return fmt.Errorf("invalid resource order item: %s. allowed are: %s", item, allowed)
				}
			}
			return nil
		}).
		Value()

	// formatting
	fs.BoolVar(&cfg.NoDirSlash, "no-dir-slash", false, "Disable trailing slash for directory resources.").
		Short("D").
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
	cfg.ResourceOrder = processor.ParseResourceOrder(*order)

	return cfg, nil
}
