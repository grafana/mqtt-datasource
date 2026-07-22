package plugin

import (
	"context"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestMQTTDatasource_SubscribeStream_Security(t *testing.T) {
	ds := &MQTTDatasource{}

	tests := []struct {
		name           string
		requestPath    string
		userNamespace  string
		expectedStatus backend.SubscribeStreamStatus
		expectError    bool
	}{
		{
			name:           "valid namespace matches",
			requestPath:    "ds/uid123/1s/sensor/temp/datasource-uid/hash123/stacks-456",
			userNamespace:  "stacks-456",
			expectedStatus: backend.SubscribeStreamStatusOK,
			expectError:    false,
		},
		{
			name:           "invalid namespace mismatch",
			requestPath:    "ds/uid123/1s/sensor/temp/datasource-uid/hash123/stacks-456",
			userNamespace:  "stacks-789",
			expectedStatus: backend.SubscribeStreamStatusPermissionDenied,
			expectError:    true,
		},
		{
			name:           "invalid path format - too short",
			requestPath:    "ds/uid123/1s",
			userNamespace:  "stacks-456",
			expectedStatus: backend.SubscribeStreamStatusNotFound,
			expectError:    true,
		},
		{
			name:           "different user same namespace - should work",
			requestPath:    "ds/uid123/1s/sensor/temp/datasource-uid/different-hash/stacks-456",
			userNamespace:  "stacks-456",
			expectedStatus: backend.SubscribeStreamStatusOK,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context with plugin config
			pCtx := backend.PluginContext{
				Namespace: tt.userNamespace,
			}
			ctx := backend.WithPluginContext(context.Background(), pCtx)

			req := &backend.SubscribeStreamRequest{
				Path: tt.requestPath,
			}

			resp, err := ds.SubscribeStream(ctx, req)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}

			if resp.Status != tt.expectedStatus {
				t.Errorf("Expected status %v, got %v", tt.expectedStatus, resp.Status)
			}
		})
	}
}

func TestMQTTDatasource_SubscribeStream_PathParsing(t *testing.T) {
	ds := &MQTTDatasource{}

	tests := []struct {
		name              string
		requestPath       string
		expectedNamespace string
	}{
		{
			name:              "simple streaming key",
			requestPath:       "ds/uid123/1s/sensor/temp/datasource-uid/hash123/stacks-456",
			expectedNamespace: "stacks-456",
		},
		{
			name:              "complex topic path",
			requestPath:       "ds/uid123/5s/building/floor1/room2/sensor/temp/datasource-uid/hash456/stacks-789",
			expectedNamespace: "stacks-789",
		},
		{
			name:              "streaming key with multiple segments",
			requestPath:       "ds/uid123/10s/simple/topic/my-datasource/complex-hash-value/stacks-123",
			expectedNamespace: "stacks-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context with matching org
			namespace := tt.expectedNamespace

			pCtx := backend.PluginContext{
				Namespace: namespace,
			}
			ctx := backend.WithPluginContext(context.Background(), pCtx)

			req := &backend.SubscribeStreamRequest{
				Path: tt.requestPath,
			}

			resp, err := ds.SubscribeStream(ctx, req)

			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if resp.Status != backend.SubscribeStreamStatusOK {
				t.Errorf("Expected OK status, got %v", resp.Status)
			}
		})
	}
}

func TestMQTTDatasource_SubscribeStream_EdgeCases(t *testing.T) {
	ds := &MQTTDatasource{}

	tests := []struct {
		name           string
		requestPath    string
		userNamespace  string
		expectedStatus backend.SubscribeStreamStatus
	}{
		{
			name:           "empty path",
			requestPath:    "",
			userNamespace:  "stacks-456",
			expectedStatus: backend.SubscribeStreamStatusNotFound,
		},
		{
			name:           "path with only prefix",
			requestPath:    "ds/uid123",
			userNamespace:  "stacks-456",
			expectedStatus: backend.SubscribeStreamStatusNotFound,
		},
		{
			name:           "very long path",
			requestPath:    "ds/uid123/1s/very/long/topic/path/with/many/segments/datasource-uid/hash123/stacks-456",
			userNamespace:  "stacks-456",
			expectedStatus: backend.SubscribeStreamStatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pCtx := backend.PluginContext{
				Namespace: tt.userNamespace,
			}
			ctx := backend.WithPluginContext(context.Background(), pCtx)

			req := &backend.SubscribeStreamRequest{
				Path: tt.requestPath,
			}

			resp, _ := ds.SubscribeStream(ctx, req)

			if resp.Status != tt.expectedStatus {
				t.Errorf("Expected status %v, got %v", tt.expectedStatus, resp.Status)
			}
		})
	}
}

// Test that demonstrates the security model
func TestMQTTDatasource_SubscribeStream_MultiTenantSecurity(t *testing.T) {
	ds := &MQTTDatasource{}

	// Same topic, same streaming key structure, but different orgs
	basePath := "ds/uid123/1s/sensor/temp/datasource-uid/hash123/"

	// User from namespace stacks-456 tries to access their own data - should work
	pCtx456 := backend.PluginContext{Namespace: "stacks-456"}
	ctx456 := backend.WithPluginContext(context.Background(), pCtx456)
	req456 := &backend.SubscribeStreamRequest{Path: basePath + "stacks-456"}

	resp456, err456 := ds.SubscribeStream(ctx456, req456)
	if err456 != nil {
		t.Errorf("Expected no error for valid org access, got: %v", err456)
	}
	if resp456.Status != backend.SubscribeStreamStatusOK {
		t.Errorf("Expected OK status for valid org access, got: %v", resp456.Status)
	}

	// User from namespace stacks-456 tries to access namespace stacks-789's data - should fail
	req789Data := &backend.SubscribeStreamRequest{Path: basePath + "stacks-789"}

	resp789, err789 := ds.SubscribeStream(ctx456, req789Data)
	if err789 == nil {
		t.Error("Expected error for cross-org access attempt")
	}
	if resp789.Status != backend.SubscribeStreamStatusPermissionDenied {
		t.Errorf("Expected PermissionDenied status for cross-org access, got: %v", resp789.Status)
	}

	// User from namespace stacks-789 tries to access their own data - should work
	pCtx789 := backend.PluginContext{Namespace: "stacks-789"}
	ctx789 := backend.WithPluginContext(context.Background(), pCtx789)

	resp789Own, err789Own := ds.SubscribeStream(ctx789, req789Data)
	if err789Own != nil {
		t.Errorf("Expected no error for valid org access, got: %v", err789Own)
	}
	if resp789Own.Status != backend.SubscribeStreamStatusOK {
		t.Errorf("Expected OK status for valid org access, got: %v", resp789Own.Status)
	}
}
