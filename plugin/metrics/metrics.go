package metrics

import (
	"github.com/micro/micro/v2/plugin"
)

//NewPlugin of metrics
func NewPlugin(opts ...Option) plugin.Plugin {
	return newPrometheus(opts...)
}
