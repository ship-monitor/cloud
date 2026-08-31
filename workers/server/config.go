package server

import (
	"net"
	"strconv"
	"time"

	"github.com/ship-monitor/cloud/pkg/middleware"
)

type Config struct {
	CORS              middleware.CORSConfig
	ReadHeaderTimeout time.Duration
	Port              int
}

func (c Config) Host() string {
	return net.JoinHostPort(
		"",
		strconv.Itoa(c.Port),
	)
}
