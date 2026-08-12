package actions

import (
	"fmt"
	"time"
)

// Source simulates a data source with a controllable latency. It is the
// benchmark's "50ms source" (and the 20/50/200ms nested-Suspense sources),
// exercising the real Go↔JS bridge from a Server Component.
//
// In JS:  import { Source } from "@pola/actions"; await Source.get(50)
type Source struct{}

// Value is the payload returned by Source.get.
type Value struct {
	Text string `json:"text"`
}

// Get sleeps for `ms` milliseconds, then returns a stable string. Becomes
// Source.get(ms) in JS.
func (s *Source) Get(ms int) (*Value, error) {
	if ms < 0 {
		ms = 0
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
	return &Value{Text: fmt.Sprintf("Loaded after %dms", ms)}, nil
}
