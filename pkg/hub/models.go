// Package hub provides models for interconection with hub.
//
// MAYBE: Rename package
package hub

import "time"

type Heartbeat struct {
	Timestamp time.Time `json:"timestamp"`
}
