package idrac

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/iulianpascalau/mx-import-db-orchestrator/internal/common"
	"github.com/stretchr/testify/require"
)

func TestGetPowerState(t *testing.T) {
	tests := []struct {
		name          string
		responseCode  int
		responseBody  string
		expectedState common.PowerState
		expectError   bool
	}{
		{
			name:          "Power On",
			responseCode:  http.StatusOK,
			responseBody:  `{"PowerState": "On"}`,
			expectedState: common.PowerStateOn,
			expectError:   false,
		},
		{
			name:          "Power Off",
			responseCode:  http.StatusOK,
			responseBody:  `{"PowerState": "Off"}`,
			expectedState: common.PowerStateOff,
			expectError:   false,
		},
		{
			name:          "Unknown State",
			responseCode:  http.StatusOK,
			responseBody:  `{"PowerState": "Unknown"}`,
			expectedState: "",
			expectError:   true,
		},
		{
			name:          "Error Status",
			responseCode:  http.StatusInternalServerError,
			responseBody:  ``,
			expectedState: "",
			expectError:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/redfish/v1/Systems/System.Embedded.1", r.URL.Path)
				require.Equal(t, http.MethodGet, r.Method)

				w.WriteHeader(tc.responseCode)
				_, _ = w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			client := NewClient(ClientConfig{
				BaseURL:            server.URL,
				Username:           "testuser",
				Password:           "testpass",
				InsecureSkipVerify: true,
				Timeout:            1 * time.Second,
			})

			state, err := client.GetPowerState(context.Background())

			if (err != nil) != tc.expectError {
				require.Fail(t, "Expected error to be %v, got %v", tc.expectError, err)
			}

			require.Equal(t, tc.expectedState, state)
		})
	}
}

func TestPowerOn(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/redfish/v1/Systems/System.Embedded.1/Actions/ComputerSystem.Reset", r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)

		var reqBody map[string]string
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		require.Equal(t, "On", reqBody["ResetType"])

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(ClientConfig{
		BaseURL:            server.URL,
		Username:           "testuser",
		Password:           "testpass",
		InsecureSkipVerify: true,
		Timeout:            1 * time.Second,
	})

	err := client.PowerOn(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestPowerOff(t *testing.T) {
	tests := []struct {
		name          string
		graceful      bool
		expectedReset string
	}{
		{
			name:          "Graceful Shutdown",
			graceful:      true,
			expectedReset: "GracefulShutdown",
		},
		{
			name:          "Force Off",
			graceful:      false,
			expectedReset: "ForceOff",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/redfish/v1/Systems/System.Embedded.1/Actions/ComputerSystem.Reset", r.URL.Path)
				require.Equal(t, http.MethodPost, r.Method)

				var reqBody map[string]string
				err := json.NewDecoder(r.Body).Decode(&reqBody)
				require.Nil(t, err)

				require.Equal(t, tc.expectedReset, reqBody["ResetType"])
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			client := NewClient(ClientConfig{
				BaseURL:            server.URL,
				Username:           "testuser",
				Password:           "testpass",
				InsecureSkipVerify: true,
				Timeout:            1 * time.Second,
			})

			err := client.PowerOff(context.Background(), tc.graceful)
			require.Nil(t, err)
		})
	}
}

func TestGetStatus_FunctionalTest(t *testing.T) {
	idracURL := os.Getenv("IDRAC_URL")
	idracUser := os.Getenv("IDRAC_USER")
	idracPass := os.Getenv("IDRAC_PASS")

	if len(idracURL) == 0 || len(idracUser) == 0 || len(idracPass) == 0 {
		t.Skip("this is a functional test, will need values credentials. Please define your environment variables IDRAC_URL, IDRAC_USER and IDRAC_PASS so this test can work")
	}

	client := NewClient(ClientConfig{
		BaseURL:            idracURL,
		Username:           idracUser,
		Password:           idracPass,
		InsecureSkipVerify: true,
	})

	state, err := client.GetPowerState(context.Background())
	require.Nil(t, err)

	fmt.Printf("Status got: %s\n", state)
}
