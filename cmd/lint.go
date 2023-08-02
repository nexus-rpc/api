package main

import (
	"context"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

func lintCmd() *cobra.Command {
	var l linter
	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Lint proto files",
		Run: func(cmd *cobra.Command, args []string) {
			if err := l.lint(cmd.Context()); err != nil {
				l.logger.Fatal(err)
			}
		},
	}
	l.addCLIFlags(cmd.Flags())
	return cmd
}

type linter struct {
	logger         *zap.SugaredLogger
	loggingOptions LoggingOptions
}

func (l *linter) addCLIFlags(fs *pflag.FlagSet) {
	l.loggingOptions.AddCLIFlags(fs)
}

func (l *linter) lint(ctx context.Context) error {
	l.logger = l.loggingOptions.MustCreateLogger()
	protoFiles, err := findProtos()
	if err != nil {
		return err
	}
	args := append([]string{"--set-exit-status", "--config", "./api-linter.yaml", "-I", "."}, protoFiles...)
	l.logger.Infow("Running api-linter", "args", args)
	cmd := exec.CommandContext(ctx, "api-linter", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	l.logger.Infow("Running buf lint")
	cmd = exec.CommandContext(ctx, "buf", "lint")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
