# Caddy `wsproxy` plugin

This plugin act as a bridge between web sockets and TCP sockets to enable frontend apps to access to TCP sockets through web sockets.


## Installation

You can add the plugin to caddy source manually or using [caddyman](https://github.com/incubaid/caddyman/)

##### 1 - Manual Installation
* Get plugin files:
```bash
go get github.com/arahmanhamdy/wsproxy
```
* Import plugin into caddy `run.go` file: `$GOPATH/src/github.com/mholt/caddy/caddy/caddymain/run.go`
```go
package caddymain
import (
	//.......
	
	// This is where other plugins get plugged in (imported)
	// ......
	_ "github.com/arahmanhamdy/wsproxy"
)
```
* Register `wsproxy` directive by adding it into `plugin.go` file: `$GOPATH/src/github.com/mholt/caddy/caddyhttp/httpserver/plugin.go`
```go
package httpserver
// .......
var directives = []string{
	//........
	"wsproxy",
	//............
}
```
* Rebuild Caddy
```bash
cd $GOPATH/src/github.com/mholt/caddy/caddy/
go run build.go
```

##### 2 - Using Caddyman
```bash
git clone https://github.com/Incubaid/caddyman.git
cd caddman
./caddyman.sh install wsproxy
```

## Plugin Usage
In your caddy file use the `wsproxy` directive:

`wsproxy [PATH] TCP_SOCKET_ADDRESS`

where: 

`PATH` is the server path which will start websocket connection, if ommited it will match `/`

`TCP_SOCKET_ADDRESS` is the tcp socket address needs to connect to.

Example:
```
http://localhost:8200 {
    wsproxy     /redis      localhost:6379
}
```
