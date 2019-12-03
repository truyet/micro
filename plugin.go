package main

import (
	"io"
)

var (
	apiTracerCloser, webTracerCloser io.Closer
)

func cleanWork() error {
	// closer
	webTracerCloser.Close()
	apiTracerCloser.Close()

	return nil
}

// 插件注册
func init() {

}
