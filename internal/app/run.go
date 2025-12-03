package app

import (
	"context"
	"fmt"
	"io"

	"github.com/gi8lino/karma/internal/cli"
	"github.com/gi8lino/karma/internal/logging"
	"github.com/gi8lino/karma/internal/processor"

	"github.com/containeroo/tinyflags"
)

// Run wires parsing, logging, and processing to execute the command.
func Run(ctx context.Context, version string, args []string, stdOut, stdErr io.Writer) error {
// Parse the CLI flags.
	cfg, err := cli.Parse(version, args)
	if err != nil {
		if tinyflags.IsHelpRequested(err) || tinyflags.IsVersionRequested(err) {
			fmt.Fprint(stdOut, err.Error()) // nolint:errcheck
			return nil
		}
		return fmt.Errorf("CLI flags error: %w", err)
	}

// Set up the logger.
	logLevel := logging.LevelFromVerbosity(cfg.Verbosity)
	logger := logging.New(stdOut, stdErr, logLevel)

// Log the version and configuration.
	logger.DebugKV("version", version)
	logger.DebugKV(
		"skip", fmt.Sprintf("%v", cfg.SkipPatterns),
		"gitignore", fmt.Sprintf("%v", cfg.GitIgnore),
		"include-dot", fmt.Sprintf("%v", cfg.IncludeDot),
		"dir-suffix", fmt.Sprintf("%v", cfg.AddDirSuffix),
		"dir-prefix", fmt.Sprintf("%v", cfg.AddDirPrefix),
		"ignored-prefixes", fmt.Sprintf("%v", cfg.IgnoredPrefixes),
		"order", fmt.Sprintf("%v", cfg.ResourceOrder),
	)

// Create the processor options.
	opts := processor.Options{
		Skip:            cfg.SkipPatterns,
		UseGitIgnore:    cfg.GitIgnore,
		IncludeDot:      cfg.IncludeDot,
		AddDirSuffix:    cfg.AddDirSuffix,
		AddDirPrefix:    cfg.AddDirPrefix,
		IgnoredPrefixes: cfg.IgnoredPrefixes,
		ResourceOrder:   cfg.ResourceOrder,
	}

// Process each base directory.
	var totalStats processor.ResourceStats
	for _, dir := range cfg.BaseDirs {
		logger.Processing("base", "path", dir)
		proc := processor.New(opts, logger)
		stats, err := proc.Process(ctx, dir)
		if err != nil {
			return err
		}
		totalStats.Add(stats)
	}

// Print the summary.
	logger.Summary(
		totalStats.Updated,
		totalStats.NoOp,
		totalStats.Reordered,
		totalStats.Added,
		totalStats.Removed,
	)

	return nil
}
