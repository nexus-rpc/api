package main

import (
	"context"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

var deps = []string{
	"github.com/bufbuild/buf/cmd/buf@v1.25.1",
	"github.com/googleapis/api-linter/cmd/api-linter@v1.55.2",
	"google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0",
}

func installDepsCmd() *cobra.Command {
	var i depsInstaller
	cmd := &cobra.Command{
		Use:   "install-deps",
		Short: "Install tool dependencies",
		Run: func(cmd *cobra.Command, args []string) {
			if err := i.installDeps(cmd.Context()); err != nil {
				i.logger.Fatal(err)
			}
		},
	}
	i.addCLIFlags(cmd.Flags())
	return cmd
}

type depsInstaller struct {
	logger         *zap.SugaredLogger
	loggingOptions LoggingOptions
}

func (l *depsInstaller) addCLIFlags(fs *pflag.FlagSet) {
	l.loggingOptions.AddCLIFlags(fs)
}

func (l *depsInstaller) installDeps(ctx context.Context) error {
	l.logger = l.loggingOptions.MustCreateLogger()

	for _, dep := range deps {
		args := []string{"install", dep}

		l.logger.Infow("Installing dependency", "dep", dep)
		cmd := exec.CommandContext(ctx, "go", args...)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}
