// Package hub provides models for interconection with hub.
//
// MAYBE: Rename package
package hub

import "time"

type Heartbeat struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
}
