package tests

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

func invoke(t *testing.T, name string, query string, expectedStatusCode ...int) []byte {
	return invokeWithVerb(t, http.MethodPost, name, query, expectedStatusCode...)
}

func invokeWithVerb(t *testing.T, verb string, name string, query string, expectedStatusCode ...int) []byte {
	attempts := 30 // i.e. 30x2s = 1m
	delay := time.Millisecond * 750

	breakoutStatus := []int{http.StatusUnauthorized}

	gwURL, urlErr := url.Parse(os.Getenv("gateway_url"))
	if urlErr != nil {
		t.Log(urlErr)
		t.Fail()
	}
	gwURL.Path = fmt.Sprintf("/function/%s", name)

	if len(query) > 0 {
		gwURL.RawQuery = query
	}

	for i := 0; i < attempts; i++ {

		bytesOut, res, err := httpReq(gwURL.String(), verb, nil)

		if err != nil {
			t.Log(err.Error())
			t.Fail()
		}

		validMatch := false
		for _, code := range expectedStatusCode {
			if res.StatusCode == code {
				validMatch = true
				break
			}
		}

		breakout := false

		for _, code := range breakoutStatus {
			if res.StatusCode == code {
				breakout = true
				break
			}
		}

		if !validMatch {
			t.Logf(
				"[%d/%d] Bad response, got: %d - %s, but want: %v",
				i+1,
				attempts,
				res.StatusCode,
				gwURL.String(),
				expectedStatusCode,
			)

			if breakout {
				t.Logf("Received breakout-status %d, failing test", res.StatusCode)
				t.Fail()
				return bytesOut
			}

			if i == attempts-1 {

				t.Logf("Failing after: %d attempts", attempts)
				t.Logf(string(bytesOut))
				t.Fail()
			}
			time.Sleep(delay)
			continue
		}

		if attempts > 0 {
			t.Logf(
				"[%d/%d] Got correct response: %v - %s",
				i+1, attempts,
				res.StatusCode,
				gwURL.String(),
			)
		}

		return bytesOut
	}
	return nil
}
