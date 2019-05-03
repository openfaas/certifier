package tests

import (
	"net/http"
	"os"
	"testing"
)

func Test_HealthEndpoint(t *testing.T) {
	_, res, err := httpReq(os.Getenv("gateway_url")+"/healthz", http.MethodGet, nil)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	if res.StatusCode != http.StatusOK {
		t.Logf("got %d, wanted %d", res.StatusCode, http.StatusOK)
		t.Fail()
	}
}