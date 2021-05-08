package tests

import (
	"context"
	"testing"
	// please only use the gateway types in this test
	// other other tests should use the provider types
)

func Test_ProviderInfo(t *testing.T) {
	systeminfo, err := config.Client.GetSystemInfo(context.Background())

	if err != nil {
		t.Fatal(err)
	}

	if systeminfo.Provider == nil {
		t.Fatal("provider info should be present")
	}
	if systeminfo.Provider.Orchestration == "" {
		t.Fatal("provider orchestration name may not be empty")
	}
	if systeminfo.Provider.Name == "" {
		t.Fatal("provider name may not be empty")
	}

	pv := systeminfo.Provider.Version
	if pv == nil {
		t.Fatal("provider version cannot be empty")
	}

	if pv.Release == "" {
		t.Fatal("provider version release may not be empty")
	}
	if pv.SHA == "" {
		t.Fatal("provider version sha may not be empty")
	}

	v := systeminfo.Version
	if v == nil {
		t.Fatal("gateway version may not be nil")
	}
	if v.Release == "" {
		t.Fatal("gateway version release may not be empty")
	}
	if v.SHA == "" {
		t.Fatal("gateway version sha may not be empty")
	}
}
