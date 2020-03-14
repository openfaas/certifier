package tests

import (
	"net/http"
	"testing"
)

func Test_HealthEndpoint(t *testing.T) {
	gwURL := gatewayUrl(t, "healthz", "")
	_, res := request(t, gwURL, http.MethodGet, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("error with /healthz, got %d, but want %d", res.StatusCode, http.StatusOK)
	}
}
