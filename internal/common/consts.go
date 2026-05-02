package common

// PowerState represents the current power status of the server
type PowerState string

const (
	PowerStateOn  PowerState = "On"
	PowerStateOff PowerState = "Off"
)
