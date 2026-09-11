package app

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/gi8lino/karma/internal/cli"
	"github.com/gi8lino/karma/internal/logging"
	"github.com/gi8lino/karma/internal/processor"

	"github.com/containeroo/tinyflags"
)

// ErrCheckFailed indicates that --check found kustomizations that need updates.
var ErrCheckFailed = errors.New("kustomizations are not in sync")

// Run wires parsing, logging, and processing to execute the command.
func Run(ctx context.Context, version string, args []string, stdOut io.Writer) error {
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
	logger := logging.New(stdOut, logLevel)

	// Log the version and configuration.
	logger.DebugKV("version", version)
	logger.DebugKV(
		"skip", fmt.Sprintf("%v", cfg.SkipPatterns),
		"opaque", fmt.Sprintf("%v", cfg.OpaquePatterns),
		"preserve", fmt.Sprintf("%v", cfg.PreservePatterns),
		"gitignore", fmt.Sprintf("%v", cfg.UseGitIgnore),
		"include-dot", fmt.Sprintf("%v", cfg.IncludeDot),
		"dir-suffix", fmt.Sprintf("%v", cfg.AddDirSuffix),
		"dir-prefix", fmt.Sprintf("%v", cfg.AddDirPrefix),
		"dry-run", fmt.Sprintf("%v", cfg.DryRun || cfg.Check),
		"check", fmt.Sprintf("%v", cfg.Check),
		"order", fmt.Sprintf("%v", cfg.ResourceOrder),
	)

	// Create the processor options.
	opts := processor.Options{
		Skip:          cfg.SkipPatterns,
		Opaque:        cfg.OpaquePatterns,
		Preserve:      cfg.PreservePatterns,
		UseGitIgnore:  cfg.UseGitIgnore,
		IncludeDot:    cfg.IncludeDot,
		AddDirSuffix:  cfg.AddDirSuffix,
		AddDirPrefix:  cfg.AddDirPrefix,
		DryRun:        cfg.DryRun || cfg.Check,
		ResourceOrder: cfg.ResourceOrder,
	}

	// Process each base directory with the same immutable processor configuration.
	proc := processor.New(opts, logger)
	var totalStats processor.ResourceStats
	for _, dir := range cfg.BaseDirs {
		logger.Processing("base", "path", dir)
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

	if cfg.Check && totalStats.Updated > 0 {
		return ErrCheckFailed
	}

	return nil
}
