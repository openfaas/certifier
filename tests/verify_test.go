package tests

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"testing"
	"time"

	sdk "github.com/openfaas/faas-cli/proxy"
)

func invoke(t *testing.T, function *sdk.DeployFunctionSpec, query string, body string, expectedStatusCode ...int) []byte {
	t.Helper()
	content, _ := invokeWithVerb(t, http.MethodPost, function, query, body, expectedStatusCode...)
	return content
}

func invokeWithVerb(t *testing.T, verb string, function *sdk.DeployFunctionSpec, query string, body string, expectedStatusCode ...int) ([]byte, *http.Response) {
	t.Helper()

	attempts := 30 // i.e. 30x2s = 1m
	delay := time.Millisecond * 750

	breakoutStatus := []int{http.StatusUnauthorized}

	uri := resourceURL(t, path.Join("function", fmt.Sprintf("%s.%s", function.FunctionName, function.Namespace)), query)

	var requestBody io.Reader
	if body != "" {
		requestBody = strings.NewReader(body)
	}

	var bytesOut []byte
	for i := 0; i < attempts; i++ {

		bytesOut, res := request(t, uri, verb, nil, requestBody)

		for _, code := range expectedStatusCode {
			if res.StatusCode == code {
				// success, we can stop now
				t.Logf("[%d/%d] Got correct response: %v - %s", i+1, attempts, res.StatusCode, uri)
				return bytesOut, res
			}
		}

		// handle fatal errors that we can not retry
		for _, code := range breakoutStatus {
			if res.StatusCode == code {
				t.Fatalf("Received breakout-status %d, invoke failed with: %s", res.StatusCode, bytesOut)
			}
		}

		// finally, log an the error attempt and wait to retry
		t.Logf("[%d/%d] Bad response want: %v, got: %d - %s", i+1, attempts, expectedStatusCode, res.StatusCode, uri)
		time.Sleep(delay)
	}

	// loop ended without success
	t.Logf("Failing after: %d attempts", attempts)
	t.Fatalf("invoke failed with: %s", bytesOut)

	return nil, nil
}
