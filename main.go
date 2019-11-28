package main

import (
	"github.com/micro/cli"
	"github.com/micro/go-micro"
	"github.com/micro/micro/cmd"
	"github.com/micro/micro/plugin"
)

func main() {
	cmd.Init(
		micro.BeforeStart(gatewayCheck),
		micro.AfterStop(cleanWork),
	)
}

//gatewaycheck will restrict service run as a gateway
func gatewayCheck() error {
	plugin.Register(plugin.NewPlugin(
		plugin.WithName("restrict-func-to-gateway"),
		plugin.WithInit(func(ctx *cli.Context) error {
			return nil
		}),
	))

	return nil
}
