// Package aggregator maintains per-GPU energy state and exposes a snapshot for the exporter.
package aggregator

import (
	"context"

	"github.com/mavvrik/gpu-economics-agent/internal/models"
)

// Run consumes NodeSamples from the channel and updates the shared State.
// Exits when ctx is cancelled.
func Run(ctx context.Context, state *State, samplesIn <-chan models.NodeSample) {
	for {
		select {
		case <-ctx.Done():
			return
		case sample, ok := <-samplesIn:
			if !ok {
				return
			}
			state.Update(sample)
		}
	}
}
