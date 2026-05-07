package proxmox

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestIsRunning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api2/json/version", r.URL.Path)
		require.Equal(t, "PVEAPIToken=testuser@pve!token=123", r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL:            server.URL,
		APIToken:           "testuser@pve!token=123",
		InsecureSkipVerify: true,
		Timeout:            1 * time.Second,
	})

	running := client.IsRunning(context.Background())
	require.True(t, running)
}

func TestIsRunning_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL:            server.URL,
		APIToken:           "token",
		InsecureSkipVerify: true,
		Timeout:            1 * time.Second,
	})

	running := client.IsRunning(context.Background())
	require.False(t, running)
}

func TestGetVirtualMachines_WithTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api2/json/cluster/resources", r.URL.Path)
		require.Equal(t, "type=vm", r.URL.RawQuery)

		response := `{
			"data": [
				{
					"vmid": 100,
					"name": "node-1",
					"node": "pve1",
					"type": "qemu",
					"status": "running",
					"tags": "import-db,shard-0"
				},
				{
					"vmid": 101,
					"name": "node-2",
					"node": "pve1",
					"type": "lxc",
					"status": "stopped",
					"tags": "epochs-0-100; another-tag"
				},
				{
					"vmid": 102,
					"name": "not-a-vm",
					"node": "pve1",
					"type": "storage",
					"status": "unknown"
				}
			]
		}`
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL:            server.URL,
		APIToken:           "token",
		InsecureSkipVerify: true,
		Timeout:            1 * time.Second,
	})

	vms, err := client.GetVirtualMachines(context.Background())
	require.NoError(t, err)
	require.Len(t, vms, 2) // Should filter out "storage"

	require.Equal(t, 100, vms[0].VMID)
	require.Equal(t, "qemu", vms[0].Type)
	require.Equal(t, []string{"import-db", "shard-0"}, vms[0].Tags)

	require.Equal(t, 101, vms[1].VMID)
	require.Equal(t, "lxc", vms[1].Type)
	require.Equal(t, []string{"epochs-0-100", "another-tag"}, vms[1].Tags)
}

func TestGetVirtualMachines_FallbackTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api2/json/cluster/resources" {
			response := `{
				"data": [
					{
						"vmid": 100,
						"name": "node-1",
						"node": "pve1",
						"type": "qemu",
						"status": "running"
					}
				]
			}`
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(response))
			return
		}

		if r.URL.Path == "/api2/json/nodes/pve1/qemu/100/config" {
			response := `{
				"data": {
					"tags": "import-db, shard-1"
				}
			}`
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(response))
			return
		}

		t.Fatalf("Unexpected request path: %s", r.URL.Path)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL:            server.URL,
		APIToken:           "token",
		InsecureSkipVerify: true,
		Timeout:            1 * time.Second,
	})

	vms, err := client.GetVirtualMachines(context.Background())
	require.NoError(t, err)
	require.Len(t, vms, 1)

	require.Equal(t, 100, vms[0].VMID)
	require.Equal(t, []string{"import-db", "shard-1"}, vms[0].Tags)
}

func TestGetStatus_FunctionalTest(t *testing.T) {
	//proxmoxURL in the ip:port format
	// ecample: 192.168.1.2:8006
	proxmoxURL := os.Getenv("PROXMOX_URL")
	//proxmoxToken in this format: [User]@[Realm]![Token ID]=[Token Secret]
	// example: root@pam!orchestrator=12345678-abcd-1234-abcd-123456789abc
	proxmoxToken := os.Getenv("PROXMOX_TOKEN")

	if len(proxmoxURL) == 0 || len(proxmoxToken) == 0 {
		t.Skip("this is a functional test, will need values credentials. Please define your environment variables PROXMOX_URL and PROXMOX_TOKEN so this test can work")
	}

	client := NewClient(ClientConfig{
		BaseURL:            proxmoxURL,
		APIToken:           proxmoxToken,
		InsecureSkipVerify: true,
	})

	isRunning := client.IsRunning(context.Background())

	fmt.Printf("Is running? %v\n", isRunning)

	if !isRunning {
		fmt.Println("Proxmox is not running, can not get VMs")
		return
	}

	vms, err := client.GetVirtualMachines(context.Background())
	require.Nil(t, err)

	for _, vm := range vms {
		fmt.Printf("VM %d (on %s)\n\t name: %s, status: %s, type: %s, tags: [%s] \n",
			vm.VMID, vm.Node, vm.Name, vm.Status, vm.Type, strings.Join(vm.Tags, ", "))
	}
}
