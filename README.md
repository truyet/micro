# Micro  API

## Overview

注意China go mod加速问题：
[官方讨论](https://github.com/golang/go/issues/31755)

关联问题：
[Go Modules 和 Proxy](https://github.com/guanhui07/blog/issues/642)
[Go Module China 加速](https://github.com/developer-learning/night-reading-go/issues/468)

设置参考：

```bash
go env -w GOPROXY=https://goproxy.cn,direct   //go >= 1.13
go env -w GOSUMDB=sum.golang.org

或者是
go env -w GOPROXY=direct
go env -w GOSUMDB=sum.golang.google.cn
```

## enable go modules

export GO111MODULE=on

## Usage

See all the options

```bash
micro --help
```

See the [docs](https://micro.mu/docs/) for detailed information on the architecture, installation and use of the platform.
