package tests

import (
	"context"
	"testing"

	// please only use the gateway types in this test
	// other other tests should use the provider types
	sdk "github.com/openfaas/faas-cli/proxy"
)

func Test_ProviderInfo(t *testing.T) {
	gwURL := gatewayURL(t)

	client := sdk.NewClient(&Unauthenticated{}, gwURL, nil, &timeout)
	systeminfo, err := client.GetSystemInfo(context.Background())

	if err != nil {
		t.Fatal(err)
	}

	p, ok := systeminfo["provider"]
	if !ok {
		t.Fatal("provider info should be present")
	}
	provider := p.(map[string]interface{})

	if orch, ok := provider["orchestration"]; !ok || orch.(string) == "" {
		t.Fatal("provider orchestration name may not be empty")
	}

	if name, ok := provider["provider"]; !ok || name.(string) == "" {
		t.Fatal("provider name may not be empty")
	}

	pv, ok := provider["version"]
	if !ok {
		t.Fatal("provider version cannot be empty")
	}
	providerVersion := pv.(map[string]interface{})
	if release, ok := providerVersion["release"]; !ok || release.(string) == "" {
		t.Fatal("provider version release may not be empty")
	}
	if sha, ok := providerVersion["sha"]; !ok || sha.(string) == "" {
		t.Fatal("provider version sha may not be empty")
	}

	v, ok := systeminfo["version"]
	version := v.(map[string]interface{})

	if !ok {
		t.Fatal("gateway version may not be nil")
	}

	if release, ok := version["release"]; !ok || release.(string) == "" {
		t.Fatal("gateway version release may not be empty")
	}

	if sha, ok := version["sha"]; !ok || sha.(string) == "" {
		t.Fatal("gateway version sha may not be empty")
	}

}
