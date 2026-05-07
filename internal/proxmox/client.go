package proxmox

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/iulianpascalau/mx-import-db-orchestrator/internal/common"
)

type proxmoxClient struct {
	httpClient *http.Client
	baseURL    string
	apiToken   string // Format: USER@REALM!TOKENID=UUID
}

// ClientConfig holds the configuration for the Proxmox client
type ClientConfig struct {
	BaseURL            string
	APIToken           string
	InsecureSkipVerify bool
	Timeout            time.Duration
}

// NewClient creates a new Proxmox client that implements the Client interface
func NewClient(config ClientConfig) *proxmoxClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
		},
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	baseURL := common.EnsureHTTPSPrefix(config.BaseURL)

	return &proxmoxClient{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
		baseURL:  baseURL,
		apiToken: config.APIToken,
	}
}

func (c *proxmoxClient) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	var reqBody []byte
	var err error
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("PVEAPIToken=%s", c.apiToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

func (c *proxmoxClient) IsRunning(ctx context.Context) bool {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api2/json/version", nil)
	if err != nil {
		return false
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

type resourceResponse struct {
	Data []struct {
		VMID   int    `json:"vmid"`
		Name   string `json:"name"`
		Node   string `json:"node"`
		Type   string `json:"type"` // qemu, lxc
		Status string `json:"status"`
		Tags   string `json:"tags"` // present in newer PVE versions
	} `json:"data"`
}

type configResponse struct {
	Data struct {
		Tags string `json:"tags"`
	} `json:"data"`
}

func (c *proxmoxClient) GetVirtualMachines(ctx context.Context) ([]VirtualMachine, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api2/json/cluster/resources?type=vm", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var res resourceResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var vms []VirtualMachine
	for _, item := range res.Data {
		// Ensure we are only grabbing VMs/LXCs
		if item.Type != "qemu" && item.Type != "lxc" {
			continue
		}

		vm := VirtualMachine{
			VMID:   item.VMID,
			Name:   item.Name,
			Node:   item.Node,
			Type:   item.Type,
			Status: item.Status,
		}

		// Try parsing tags from the resource call (works on newer PVE)
		if item.Tags != "" {
			vm.Tags = parseTags(item.Tags)
		} else {
			// If missing, try fetching from config (fallback for older PVE)
			var tags string
			tags, err = c.getVMTags(ctx, item.Node, item.Type, item.VMID)
			if err == nil && tags != "" {
				vm.Tags = parseTags(tags)
			}
		}

		vms = append(vms, vm)
	}

	return vms, nil
}

func (c *proxmoxClient) getVMTags(ctx context.Context, node, vmType string, vmid int) (string, error) {
	endpoint := fmt.Sprintf("/api2/json/nodes/%s/%s/%d/config", node, vmType, vmid)
	resp, err := c.doRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var res configResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	if err != nil {
		return "", err
	}

	return res.Data.Tags, nil
}

func parseTags(tagStr string) []string {
	if tagStr == "" {
		return nil
	}

	parts := strings.Split(tagStr, ",")
	var tags []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		// Proxmox tags also use semicolons sometimes depending on the version
		subParts := strings.Split(trimmed, ";")
		for _, sp := range subParts {
			spTrimmed := strings.TrimSpace(sp)
			if spTrimmed != "" {
				tags = append(tags, spTrimmed)
			}
		}
	}
	return tags
}
