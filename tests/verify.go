package tests

import (
	"net/http"
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

	uri := os.Getenv("gateway_url") + "function/" + name
	if len(query) > 0 {
		uri = uri + "?" + query
	}

	for i := 0; i < attempts; i++ {

		bytesOut, res, err := httpReq(uri, verb, nil)

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
			t.Logf("[%d/%d] Bad response want: %v, got: %d - %s", i+1, attempts, expectedStatusCode, res.StatusCode, uri)

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
			t.Logf("[%d/%d] Got correct response: %v - %s", i+1, attempts, res.StatusCode, uri)
		}

		return bytesOut
	}
	return nil
}
