package tests

import (
	"net/http"
	"testing"
)

func Test_HealthEndpoint(t *testing.T) {
	gwURL := resourceURL(t, "healthz", "")
	// TODO: enable auth
	_, res := request(t, gwURL, http.MethodGet, config.Auth, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("error with /healthz, got %d, but want %d", res.StatusCode, http.StatusOK)
	}
}
