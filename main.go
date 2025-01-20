package main

import (
	"gofr.dev/pkg/gofr"

	"gofr.dev/cli/gofr/bootstrap"
	"gofr.dev/cli/gofr/grpc"
	"gofr.dev/cli/gofr/migration"
)

func main() {
	cli := gofr.NewCMD()

	cli.SubCommand("init", bootstrap.Create)

	cli.SubCommand("version",
		func(*gofr.Context) (interface{}, error) {
			return CLIVersion, nil
		},
	)

	cli.SubCommand("migrate create", migration.Migrate)

	cli.SubCommand("grpc client client", grpc.BuildGRPCGoFrServer)

	cli.SubCommand("grpc client client", grpc.BuildGRPCGoFrClient)

	cli.Run()
}
