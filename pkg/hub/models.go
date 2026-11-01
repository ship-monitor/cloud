// Package hub provides models for interconection with hub.
// TODO: Rename package
package hub

import "time"

type Heartbeat struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}
