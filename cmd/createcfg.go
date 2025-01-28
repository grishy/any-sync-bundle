package cmd

import (
	"context"

	"github.com/urfave/cli/v2"
)

func GenerateCfg(ctx context.Context) *cli.Command {
	return &cli.Command{
		Name:  "generate-cfg",
		Usage: "Only generate configuration files, without starting the service",
		Action: func(cCtx *cli.Context) error {

			return nil
		},
	}
}
