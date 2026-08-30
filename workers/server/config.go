package server

import (
	"net"
	"strconv"
	"time"
)

type Config struct {
	CORS struct {
		AllowedOrigins []string
	}
	ReadHeaderTimeout time.Duration
	Port              int
}

func (c Config) Host() string {
	return net.JoinHostPort(
		"",
		strconv.Itoa(c.Port),
	)
}
