package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/s2-streamstore/s2-sdk-go/s2"
)

func TestStorageClassLifecycle(t *testing.T) {
	const basinName = "tf-storage-test"
	const streamName = "events"
	var mu sync.Mutex
	var basinExists, streamExists bool
	var basinClass, streamClass string
	location := map[string]any{
		"name": "test-location", "is_private": false,
		"storage_classes": []string{"future", "archive"}, "default_storage_class": "future",
	}
	basinInfo := map[string]any{"name": basinName, "location": "test-location"}
	streamInfo := map[string]any{"name": streamName, "created_at": "2026-01-01T00:00:00Z"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		requestBody, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var body map[string]json.RawMessage
		if len(requestBody) > 0 {
			if err := json.Unmarshal(requestBody, &body); err != nil {
				t.Errorf("decode request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
		}
		storageClass := func(raw json.RawMessage, fallback string) string {
			var config s2.StreamConfig
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, &config); err != nil {
					t.Errorf("decode stream config: %v", err)
				}
			}
			if config.StorageClass != nil {
				return *config.StorageClass
			}
			return fallback
		}
		var response any
		switch r.Method + " " + r.URL.Path {
		case "GET /v1/locations":
			response = []any{location}
		case "GET /v1/locations/default":
			response = location
		case "PUT /v1/basins/" + basinName:
			var config struct {
				DefaultStreamConfig json.RawMessage `json:"default_stream_config"`
			}
			if raw := body["config"]; len(raw) > 0 {
				if err := json.Unmarshal(raw, &config); err != nil {
					t.Errorf("decode basin config: %v", err)
				}
			}
			basinExists = true
			basinClass = storageClass(config.DefaultStreamConfig, "future")
			response = basinInfo
		case "GET /v1/basins/" + basinName:
			if !basinExists {
				w.WriteHeader(http.StatusNotFound)
				response = map[string]any{"code": "basin_not_found", "message": "not found"}
				break
			}
			response = map[string]any{"default_stream_config": map[string]any{"storage_class": basinClass}}
		case "PATCH /v1/basins/" + basinName:
			if raw, ok := body["default_stream_config"]; ok {
				basinClass = storageClass(raw, "future")
			}
			response = map[string]any{"default_stream_config": map[string]any{"storage_class": basinClass}}
		case "GET /v1/basins":
			basins := []any{}
			if basinExists {
				basins = append(basins, basinInfo)
			}
			response = map[string]any{"basins": basins, "has_more": false}
		case "DELETE /v1/basins/" + basinName:
			basinExists = false
			w.WriteHeader(http.StatusNoContent)
			return
		case "PUT /v1/streams/" + streamName:
			streamExists = true
			streamClass = storageClass(requestBody, basinClass)
			response = streamInfo
		case "GET /v1/streams/" + streamName:
			if !streamExists {
				w.WriteHeader(http.StatusNotFound)
				response = map[string]any{"code": "stream_not_found", "message": "not found"}
				break
			}
			response = map[string]any{"storage_class": streamClass}
		case "PATCH /v1/streams/" + streamName:
			if raw, ok := body["storage_class"]; ok {
				streamClass = basinClass
				if string(raw) != "null" {
					if err := json.Unmarshal(raw, &streamClass); err != nil {
						t.Errorf("decode storage class: %v", err)
					}
				}
			}
			response = map[string]any{"storage_class": streamClass}
		case "GET /v1/streams":
			streams := []any{}
			if streamExists {
				streams = append(streams, streamInfo)
			}
			response = map[string]any{"streams": streams, "has_more": false}
		case "DELETE /v1/streams/" + streamName:
			streamExists = false
			w.WriteHeader(http.StatusNoContent)
			return
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()

	config := func(basinConfig, streamConfig string) string {
		return fmt.Sprintf(`
provider "s2" {
  access_token = "test"
  account_endpoint = %q
  basin_endpoint = %q
}
data "s2_locations" "test" {}
data "s2_default_location" "test" {}
resource "s2_basin" "test" {
  name = %q
  %s
}
resource "s2_stream" "test" {
  basin = s2_basin.test.name
  name = %q
  %s
}
`, server.URL, server.URL, basinName, basinConfig, streamName, streamConfig)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config("", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("s2_basin.test", "default_stream_config.storage_class"),
					resource.TestCheckResourceAttr("s2_stream.test", "storage_class", "future"),
					resource.TestCheckResourceAttr("data.s2_locations.test", "locations.0.storage_classes.0", "future"),
					resource.TestCheckResourceAttr("data.s2_locations.test", "locations.0.default_storage_class", "future"),
					resource.TestCheckResourceAttr("data.s2_default_location.test", "storage_classes.1", "archive"),
					resource.TestCheckResourceAttr("data.s2_default_location.test", "default_storage_class", "future"),
				),
			},
			{
				Config: config(`default_stream_config { storage_class = "archive" }`, `storage_class = "archive"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("s2_basin.test", "default_stream_config.storage_class", "archive"),
					resource.TestCheckResourceAttr("s2_stream.test", "storage_class", "archive"),
				),
			},
			{
				ResourceName:                         "s2_basin.test",
				ImportState:                          true,
				ImportStateId:                        basinName,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "name",
				ImportStateVerifyIgnore: []string{
					"default_stream_config.retention_policy",
					"default_stream_config.timestamping",
					"default_stream_config.delete_on_empty",
				},
			},
			{
				Config: config("", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("s2_basin.test", "default_stream_config.storage_class"),
					resource.TestCheckResourceAttr("s2_stream.test", "storage_class", "archive"),
				),
			},
		},
	})
	mu.Lock()
	defer mu.Unlock()
	if basinClass != "future" {
		t.Fatalf("removing default_stream_config did not reset the class: got %q", basinClass)
	}
}
