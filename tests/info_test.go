package tests

import (
	"encoding/json"
	"net/http"
	"path"
	"testing"

	// please only use the gateway types in this test
	// other other tests should use the provider types
	gwtypes "github.com/openfaas/faas/gateway/types"
)

func Test_ProviderInfo(t *testing.T) {
	gwURL := gatewayUrl(t, path.Join("system", "info"), "")
	payload, resp := request(t, gwURL, http.MethodGet, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status code %d, expected 200", resp.StatusCode)
	}

	info := gwtypes.GatewayInfo{}
	err := json.Unmarshal(payload, &info)
	if err != nil {
		t.Logf(string(payload))
		t.Fatalf("unexpected error unmarshaling provider info: %s", err)
	}

	if info.Provider == nil {
		t.Fatal("provider info may not be nil")
	}

	if info.Provider.Orchestration == "" {
		t.Fatal("provider orchestration name may not be empty")
	}

	if info.Provider.Name == "" {
		t.Fatal("provider name may not be empty")
	}

	if info.Provider.Version.Release == "" {
		t.Fatal("provider version release may not be empty")
	}
	if info.Provider.Version.SHA == "" {
		t.Fatal("provider version sha may not be empty")
	}

	if info.Version == nil {
		t.Fatal("gateway version may not be nil")
	}

	if info.Version.Release == "" {
		t.Fatal("gateway version release may not be empty")
	}
	if info.Version.SHA == "" {
		t.Fatal("gateway version sha may not be empty")
	}

}
