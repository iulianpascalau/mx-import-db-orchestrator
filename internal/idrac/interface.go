package idrac

import (
	"context"

	"github.com/iulianpascalau/mx-import-db-orchestrator/internal/common"
)

// Client defines the decoupled interface for iDRAC server interactions
// TODO: move this
type Client interface {
	// GetPowerState retrieves the current power status of the server
	GetPowerState(ctx context.Context) (common.PowerState, error)

	// PowerOn sends a command to turn the server on
	PowerOn(ctx context.Context) error

	// PowerOff sends a command to turn the server off
	PowerOff(ctx context.Context, graceful bool) error
}
