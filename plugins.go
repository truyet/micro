package main

import (
	"io"
	"net/http"
	"github.com/micro/micro/api"
	"github.com/micro/micro/plugin/micro/cors"
)

var (
	apiTracerCloser, webTracerCloser io.Closer
)

func cleanwork() error {
	// closer
	webTracerCloser.Close()
	apiTracerCloser.Close()

	return nil
}

// 插件注册
func init() {
	// 跨域
	initCors()

}

func initCors() {
	// 跨域
	corsPlugin := cors.NewPlugin(
		cors.WithAllowMethods(http.MethodHead, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete),
		cors.WithAllowCredentials(true),
		cors.WithMaxAge(3600),
		cors.WithUseRsPkg(true),
	)
	api.Register(corsPlugin)
}

