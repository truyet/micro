// Copyright 2020 Asim Aslam
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Original source: github.com/micro/go-micro/v3/server/grpc/response.go

package grpc

import (
	"github.com/micro/micro/v3/platform/codec"
)

type rpcResponse struct {
	header map[string]string
	codec  codec.Codec
}

func (r *rpcResponse) Codec() codec.Writer {
	return r.codec
}

func (r *rpcResponse) WriteHeader(hdr map[string]string) {
	for k, v := range hdr {
		r.header[k] = v
	}
}

func (r *rpcResponse) Write(b []byte) error {
	return r.codec.Write(&codec.Message{
		Header: r.header,
		Body:   b,
	}, nil)
}
