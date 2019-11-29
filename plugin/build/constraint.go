package build

import (
	"log"

	"github.com/micro/cli"
	"github.com/micro/micro/plugin"
)

//ConstrainScope returns constrain mod run as gateway
func ConstrainScope() plugin.Plugin {
	return plugin.NewPlugin(
		plugin.WithName("restrict-func-to-gateway"),
		plugin.WithInit(func(ctx *cli.Context) error {
			plugins := ctx.Args()
			log.Println("sending email to:", plugins)
			//todo ....
			//ctx.Set()
			return nil
		}),
	)
}
