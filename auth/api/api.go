package api

import (
	"github.com/micro/cli/v2"
	"github.com/micro/go-micro/v2"
	"github.com/micro/go-micro/v2/util/log"
	pb "github.com/micro-in-cn/x-gateway/auth/api/proto"
)

var (
	// Name of the auth api
	Name = "go.micro.api.auth"
	// Address is the api address
	Address = ":8011"
)

// Run the micro auth api
func Run(ctx *cli.Context, srvOpts ...micro.Option) {
	log.Name("auth")

	service := micro.NewService(
		micro.Name(Name),
		micro.Address(Address),
	)

	pb.RegisterAuthHandler(service.Server(), NewHandler(service))

	if err := service.Run(); err != nil {
		log.Error(err)
	}
}
