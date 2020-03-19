package tests

import (
	"encoding/json"
	"net/http"
	"path"
	"testing"

	"github.com/openfaas/faas-provider/types"
)

func deploy(t *testing.T, createRequest types.FunctionDeployment) int {
	t.Helper()

	gwURL := gatewayUrl(t, "/system/functions", "")
	body, res := request(t, gwURL, http.MethodPost, makeReader(createRequest))
	if res.StatusCode >= 400 {
		t.Fatalf("unable to deploy function: %s", string(body))
	}

	return res.StatusCode
}

func list(t *testing.T, expectedStatusCode int) {
	gwURL := gatewayUrl(t, "/system/functions", "")
	bytesOut, res := request(t, gwURL, http.MethodGet, nil)
	if res.StatusCode != expectedStatusCode {
		t.Fatalf("got %d, wanted %d", res.StatusCode, expectedStatusCode)
	}

	functions := []types.FunctionStatus{}
	err := json.Unmarshal(bytesOut, &functions)
	if err != nil {
		t.Fatal(err)
	}
	if len(functions) == 0 {
		t.Fatal("List functions got: 0, want: > 0")
	}
}

func get(t *testing.T, name string) types.FunctionStatus {
	gwURL := gatewayUrl(t, path.Join("system", "function", name), "")
	bytesOut, res := request(t, gwURL, http.MethodGet, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("got %d, wanted %d", res.StatusCode, http.StatusOK)
	}

	function := types.FunctionStatus{}
	err := json.Unmarshal(bytesOut, &function)
	if err != nil {
		t.Fatal(err)
	}

	return function
}

func deleteFunction(t *testing.T, name string) {
	t.Helper()

	gwURL := gatewayUrl(t, "/system/functions", "")
	payload := makeReader(deleteFunctionRequest{FunctionName: name})
	_, res := request(t, gwURL, http.MethodDelete, payload)
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("delete got %d, wanted %d", res.StatusCode, http.StatusAccepted)
	}
}

func scaleFunction(t *testing.T, name string, count int) {
	t.Helper()
	gwURL := gatewayUrl(t, path.Join("system", "scale-function", name), "")
	payload := makeReader(map[string]interface{}{"service": name, "replicas": count})

	_, res := request(t, gwURL, http.MethodPost, payload)
	if res.StatusCode != http.StatusAccepted && res.StatusCode != http.StatusOK {
		t.Fatalf("scale got %d, wanted %d (or %d)", res.StatusCode, http.StatusAccepted, http.StatusOK)
	}
}
