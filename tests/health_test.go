package tests

import (
	"net/http"
	"net/url"
	"os"
	"testing"
)

func Test_HealthEndpoint(t *testing.T) {

	gwURL, urlErr := url.Parse(os.Getenv("gateway_url"))

	if urlErr != nil {
		t.Fatal(urlErr)
	}

	gwURL.Path = "/healthz"

	_, res, err := httpReq(gwURL.String(), http.MethodGet, nil)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	if res.StatusCode != http.StatusOK {
		t.Logf("error with /healthz, got %d, but want %d", res.StatusCode, http.StatusOK)
		t.Fail()
	}
}
