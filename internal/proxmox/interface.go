package proxmox

import "context"

// VirtualMachine represents a VM or LXC in Proxmox
type VirtualMachine struct {
	VMID   int      `json:"vmid"`
	Name   string   `json:"name"`
	Node   string   `json:"node"`
	Type   string   `json:"type"`   // "qemu" or "lxc"
	Status string   `json:"status"` // "running", "stopped"
	Tags   []string `json:"tags"`
}

// Client defines the interface for interacting with a Proxmox server
type Client interface {
	// IsRunning checks if the Proxmox server API is reachable and responding
	IsRunning(ctx context.Context) bool

	// GetVirtualMachines fetches all VMs and LXCs along with their tags
	GetVirtualMachines(ctx context.Context) ([]VirtualMachine, error)
}
