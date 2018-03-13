package wsproxy

import (
	"github.com/mholt/caddy"
	"github.com/mholt/caddy/caddyhttp/httpserver"
)

func init() {
	caddy.RegisterPlugin("wsproxy", caddy.Plugin{
		ServerType: "http",
		Action:     setup,
	})
}

// setup configures a new WebSocket middleware instance.
func setup(c *caddy.Controller) error {
	websocks, err := webSocketParse(c)
	if err != nil {
		return err
	}

	httpserver.GetConfig(c).AddMiddleware(func(next httpserver.Handler) httpserver.Handler {
		return WebSocket{Next: next, Sockets: websocks}
	})

	return nil
}

func webSocketParse(c *caddy.Controller) ([]Config, error) {
	var websocks []Config

	for c.Next() {
		var val, path, tcpSocketAddr string

		// Path or socket address; not sure which yet
		if !c.NextArg() {
			return nil, c.ArgErr()
		}
		val = c.Val()
			// The next argument on this line will be the TCP socket ip:port
			if c.NextArg() {
				path = val
				tcpSocketAddr = c.Val()
			} else {
				path = "/"
				tcpSocketAddr = val
			}

		websocks = append(websocks, Config{
			Path:      path,
			TCPSocketAddr: tcpSocketAddr,
		})
	}
	return websocks, nil
}
