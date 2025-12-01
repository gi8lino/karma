package app

import (
	"context"
	"fmt"
	"io"

	"github.com/gi8lino/kustomizer/internal/cli"
	"github.com/gi8lino/kustomizer/internal/logging"
	"github.com/gi8lino/kustomizer/internal/processor"

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

	logger := logging.New(stdOut, stdErr, logging.LevelFromVerbosity(cfg.Verbosity))
	defer logger.Flush()

	logger.Debug("version", version)
	logger.Debug("skip", fmt.Sprintf("%v", cfg.SkipPatterns))

	opts := processor.Options{
		Skip:         cfg.SkipPatterns,
		UseGitIgnore: !cfg.NoGitIgnore,
		IncludeDot:   cfg.IncludeDot,
		DirSlash:     !cfg.NoDirSlash,
		DirFirst:     !cfg.NoDirFirst,
		Silent:       cfg.Silent,
	}

	var totalUpdated, totalNoOp int
	for _, dir := range cfg.BaseDirs {
		logger.Processing("base", "path", dir)
		proc := processor.New(opts, logger)
		updated, noOp, err := proc.Process(ctx, dir)
		if err != nil {
			return err
		}

		totalUpdated += updated
		totalNoOp += noOp
	}

	logger.Summary(totalUpdated, totalNoOp)
	return nil
}
