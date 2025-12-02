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
	cfg, err := cli.Parse(version, args)
	if err != nil {
		if tinyflags.IsHelpRequested(err) || tinyflags.IsVersionRequested(err) {
			fmt.Fprint(stdOut, err.Error()) // nolint:errcheck
			return nil
		}
		return fmt.Errorf("CLI flags error: %w", err)
	}

	logLevel := logging.LevelFromVerbosity(cfg.Verbosity)
	logger := logging.New(stdOut, stdErr, logLevel)

	logger.Debug("settings", "version", version)
	logger.Debug("settings", "skip", fmt.Sprintf("%v", cfg.SkipPatterns))
	logger.Debug("settings", "no-gitignore", fmt.Sprintf("%v", cfg.NoGitIgnore))
	logger.Debug("settings", "include-dot", fmt.Sprintf("%v", cfg.IncludeDot))
	logger.Debug("settings", "order", fmt.Sprintf("%v", cfg.ResourceOrder))

	opts := processor.Options{
		Skip:          cfg.SkipPatterns,
		UseGitIgnore:  !cfg.NoGitIgnore,
		IncludeDot:    cfg.IncludeDot,
		DirSlash:      !cfg.NoDirSlash,
		ResourceOrder: cfg.ResourceOrder,
	}

	var totalStats processor.ResourceStats
	for _, dir := range cfg.BaseDirs {
		logger.Processing("base", "path", dir)
		proc := processor.New(opts, logger)
		stats, err := proc.Process(ctx, dir)
		if err != nil {
			return err
		}

		totalStats.Reordered += stats.Reordered
		totalStats.Added += stats.Added
		totalStats.Removed += stats.Removed
		totalStats.Updated += stats.Updated
		totalStats.NoOp += stats.NoOp
	}

	logger.Summary(totalStats.Updated, totalStats.NoOp, totalStats.Reordered, totalStats.Added, totalStats.Removed)

	return nil
}
