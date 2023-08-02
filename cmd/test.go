package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

// Test build for these languages
var langs []string = []string{"go", "java"}

func testCmd() *cobra.Command {
	var t tester
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test build proto files",
		Run: func(cmd *cobra.Command, args []string) {
			if err := t.test(cmd.Context()); err != nil {
				t.logger.Fatal(err)
			}
		},
	}
	t.addCLIFlags(cmd.Flags())
	return cmd
}

type tester struct {
	logger         *zap.SugaredLogger
	loggingOptions LoggingOptions
}

func (l *tester) addCLIFlags(fs *pflag.FlagSet) {
	l.loggingOptions.AddCLIFlags(fs)
}

func (l *tester) test(ctx context.Context) error {
	l.logger = l.loggingOptions.MustCreateLogger()
	protoFiles, err := findProtos()
	if err != nil {
		return err
	}

	tempDir, err := os.MkdirTemp("", "proto-build")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	for _, lang := range langs {
		langDir := filepath.Join(tempDir, lang)
		if err := os.Mkdir(langDir, os.ModePerm); err != nil {
			return err
		}
		args := []string{"--fatal_warnings", fmt.Sprintf("--%s_out", lang), langDir, "-I", projectRoot()}
		args = append(args, protoFiles...)

		l.logger.Infow("Running protoc", "args", args)
		cmd := exec.CommandContext(ctx, "protoc", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}
