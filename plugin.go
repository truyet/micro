package main

import (
	"io"
	"net/http"
	fileadapter	"github.com/casbin/casbin/v2/persist/file-adapter"
	//"github.com/micro/go-micro/util/log"
	"github.com/micro/micro/pkg/micro/auth"
	"github.com/micro/micro/pkg/micro/util/response"
	"github.com/micro/micro/api"
)

var (
	apiTracerCloser io.Closer
)

func cleanWork() error {
	// closer
	apiTracerCloser.Close()

	return nil
}

// 插件注册
func init() {
	// Auth
	initAuth()
}


func initAuth() {

	casb := fileadapter.NewAdapter("./conf/casbin_policy.csv")
	auth.RegisterAdapter("default", casb)


	authPlugin := auth.NewPlugin(
		auth.WithResponseHandler(response.DefaultResponseHandler),
		auth.WithSkipperFunc(func(r *http.Request) bool {
			return false
		}),
	)
	api.Register(authPlugin)

}
