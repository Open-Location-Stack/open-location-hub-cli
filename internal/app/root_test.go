package app

import (
	"testing"

	"github.com/formation-res/open-location-hub-cli/internal/openapi"
)

func TestEnvelopeLocationsDecodesBatchPayload(t *testing.T) {
	envelope := wsEnvelope{
		Payload: []any{
			map[string]any{
				"provider_id":   "tag-1",
				"provider_type": "uwb",
				"source":        "anchor-1",
				"position": map[string]any{
					"type":        "Point",
					"coordinates": []any{8.9, 52.0, 1.2},
				},
			},
			map[string]any{
				"provider_id":   "tag-2",
				"provider_type": "ble",
				"source":        "gateway-1",
				"position": map[string]any{
					"type":        "Point",
					"coordinates": []any{8.91, 52.01},
				},
			},
		},
	}

	locations, err := envelopeLocations(envelope)
	if err != nil {
		t.Fatalf("envelopeLocations returned error: %v", err)
	}
	if len(locations) != 2 {
		t.Fatalf("got %d locations, want 2", len(locations))
	}
	if locations[0].ProviderId != "tag-1" || locations[1].ProviderId != "tag-2" {
		t.Fatalf("unexpected provider IDs: %#v", locations)
	}
}

func TestTrackableWriteFromLocationIncludesProviderMetadata(t *testing.T) {
	props := openapi.ExtensionProperties{
		"upstream_hub":      "DeepHub from Flowcate",
		"upstream_provider": "deephub",
		"upstream_topic":    "location_updates",
		"ignored":           "not copied",
	}
	body := trackableWriteFromLocation(openapi.Location{
		ProviderId:   "0080E12700EAABBA",
		ProviderType: "uwb",
		Source:       "source-1",
		Properties:   &props,
	})

	if body.Name == nil || *body.Name != "uwb 0080E12700EAABBA" {
		t.Fatalf("unexpected name: %#v", body.Name)
	}
	if body.LocationProviders == nil || len(*body.LocationProviders) != 1 || (*body.LocationProviders)[0] != "0080E12700EAABBA" {
		t.Fatalf("unexpected location providers: %#v", body.LocationProviders)
	}
	if body.Properties == nil {
		t.Fatal("properties were nil")
	}
	for key, want := range map[string]any{
		"provider_id":       "0080E12700EAABBA",
		"provider_type":     "uwb",
		"source":            "source-1",
		"upstream_hub":      "DeepHub from Flowcate",
		"upstream_provider": "deephub",
		"upstream_topic":    "location_updates",
	} {
		if got := (*body.Properties)[key]; got != want {
			t.Fatalf("property %s = %#v, want %#v", key, got, want)
		}
	}
}
