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
	UseGitIgnore  bool
	IncludeDot    bool
	Mute          bool
	AddDirSuffix  bool
	AddDirPrefix  bool
	DryRun        bool
	Check         bool
	ResourceOrder []string
}

// Parse builds user configuration from CLI args.
func Parse(version string, args []string) (Config, error) {
	fs := tinyflags.NewFlagSet("karma", tinyflags.ContinueOnError)
	fs.Version(version)
	fs.RequirePositional(1)
	fs.Note("*) skip accepts `*` wildcards plus `/*` to ignore a directory's contents and " +
		"`/**` to ignore the directory while still descending into its children.")

	cfg := Config{}

	// Selection
	fs.StringSliceVar(&cfg.SkipPatterns, "skip", []string{}, "Skip resources (comma-separated). *").
		Short("s").
		Value()

	var noGitIgnore bool
	fs.BoolVar(&noGitIgnore, "no-gitignore", false, "Disable .gitignore processing.").
		Short("g").
		OneOfGroup("gitignore").
		Value()
	fs.BoolVar(&cfg.IncludeDot, "include-dot", false, "Include hidden files and directories.").
		Short("i").
		Value()

	allowed := strings.Join(processor.DefaultResourceOrder(), ", ")
	order := fs.String("order", allowed, fmt.Sprintf("Build the resource groups in the provided order. Valid groups: %s.", allowed)).
		Validate(func(v string) error {
			dro := processor.DefaultResourceOrder()
			for _, entry := range strings.Split(v, ",") {
				if entry == "" {
					continue
				}
				if !slices.Contains(dro, entry) {
					return fmt.Errorf("invalid resource order item: %s. allowed are: %s", entry, allowed)
				}
			}
			return nil
		}).
		Placeholder(strings.Join(processor.DefaultResourceOrder(), ",")).
		HideDefault().
		Value()

	// Formatting
	fs.BoolVar(&cfg.AddDirSuffix, "suffix", false, "Enable trailing slash for directory resources.").
		Short("x").
		OneOfGroup("suffix").
		Value()
	fs.BoolVar(&cfg.AddDirPrefix, "prefix", false, "Enable prefixing directories with \"./\".").
		Short("p").
		OneOfGroup("prefix").
		Value()

	// Execution
	fs.BoolVar(&cfg.DryRun, "dry-run", false, "Show changes without writing files.").
		OneOfGroup("execution").
		Value()
	fs.BoolVar(&cfg.Check, "check", false, "Exit non-zero when changes would be required without writing files.").
		OneOfGroup("execution").
		Value()

	// Logging
	fs.CounterVar(&cfg.Verbosity, "verbose", 0, "Increase verbosity. Repeat to show more details.").
		Short("v").
		HideDefault().
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
	cfg.UseGitIgnore = !noGitIgnore
	cfg.ResourceOrder = processor.ParseResourceOrder(*order)

	return cfg, nil
}
