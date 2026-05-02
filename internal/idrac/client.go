package idrac

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

const (
	urlGetState    = "/redfish/v1/Systems/System.Embedded.1"
	urlReset       = "/redfish/v1/Systems/System.Embedded.1/Actions/ComputerSystem.Reset"
	valTurnOn      = "On"
	valForceOff    = "ForceOff"
	valGracefulOff = "GracefulShutdown"
)

type redfishClient struct {
	httpClient *http.Client
	baseURL    string
	username   string
	password   string
}

// ClientConfig holds the configuration for the iDRAC client
type ClientConfig struct {
	BaseURL            string
	Username           string
	Password           string
	InsecureSkipVerify bool
	Timeout            time.Duration
}

// NewClient creates a new iDRAC client that implements the Client interface
func NewClient(config ClientConfig) *redfishClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
		},
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	baseURL := config.BaseURL
	if baseURL != "" && !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}

	return &redfishClient{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
		},
		baseURL:  baseURL,
		username: config.Username,
		password: config.Password,
	}
}

func (c *redfishClient) doRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
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

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return c.httpClient.Do(req)
}

func (c *redfishClient) GetPowerState(ctx context.Context) (common.PowerState, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, urlGetState, nil)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		PowerState string `json:"PowerState"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if result.PowerState == string(common.PowerStateOn) {
		return common.PowerStateOn, nil
	}
	if result.PowerState == string(common.PowerStateOff) {
		return common.PowerStateOff, nil
	}

	return "", fmt.Errorf("unknown power state: %s", result.PowerState)
}

type resetActionRequest struct {
	ResetType string `json:"ResetType"`
}

func (c *redfishClient) PowerOn(ctx context.Context) error {
	return c.sendResetAction(ctx, valTurnOn)
}

func (c *redfishClient) PowerOff(ctx context.Context, graceful bool) error {
	resetType := valForceOff
	if graceful {
		resetType = valGracefulOff
	}
	return c.sendResetAction(ctx, resetType)
}

func (c *redfishClient) sendResetAction(ctx context.Context, resetType string) error {
	reqBody := resetActionRequest{
		ResetType: resetType,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, urlReset, reqBody)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
