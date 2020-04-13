package tests

import (
	"context"
	"net/http"
	"path"
	"testing"

	faasSDK "github.com/openfaas/faas-cli/proxy"
	"github.com/openfaas/faas-provider/types"
)

func deploy(t *testing.T, createRequest *faasSDK.DeployFunctionSpec) int {
	t.Helper()
	gwURL := gatewayUrl(t, "", "")

	client := faasSDK.NewClient(&FaaSAuth{}, gwURL, nil, &timeout)
	statusCode := client.DeployFunction(context.Background(), createRequest)
	if statusCode >= 400 {
		t.Fatalf("unable to deploy function: %d", statusCode)
	}

	return statusCode
}

func list(t *testing.T, expectedStatusCode int) {
	gwURL := gatewayUrl(t, "", "")

	client := faasSDK.NewClient(&FaaSAuth{}, gwURL, nil, &timeout)
	functions, err := client.ListFunctions(context.Background(), defaultNamespace)
	if err != nil {
		t.Fatal(err)
	}

	if len(functions) == 0 {
		t.Fatal("List functions got: 0, want: > 0")
	}
}

func get(t *testing.T, name string) types.FunctionStatus {
	gwURL := gatewayUrl(t, "", "")

	client := faasSDK.NewClient(&FaaSAuth{}, gwURL, nil, &timeout)
	function, err := client.GetFunctionInfo(context.Background(), name, defaultNamespace)
	if err != nil {
		t.Fatal(err)
	}

	return function
}

func deleteFunction(t *testing.T, name string) {
	t.Helper()
	gwURL := gatewayUrl(t, "", "")

	client := faasSDK.NewClient(&FaaSAuth{}, gwURL, nil, &timeout)
	err := client.DeleteFunction(context.Background(), name, defaultNamespace)
	if err != nil {
		t.Fatal(err)
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
