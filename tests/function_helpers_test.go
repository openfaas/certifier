package tests

import (
	"context"
	"net/http"
	"os"
	"path"
	"testing"

	sdk "github.com/openfaas/faas-cli/proxy"
	"github.com/openfaas/faas-provider/types"
)

var devnull = os.NewFile(0, os.DevNull)

func deploy(t *testing.T, createRequest *sdk.DeployFunctionSpec) int {
	t.Helper()
	var stdout *os.File

	// suppress the sdk fmt.Println, this hides statements like this that provide no
	// useful information to the tests and clutter the output
	// Deployed. 202 Accepted.
	// URL: http://127.0.0.1:8080/function/test-throughput-scaling
	stdout, os.Stdout = os.Stdout, devnull
	defer func() {
		os.Stdout = stdout
	}()

	statusCode := config.Client.DeployFunction(context.Background(), createRequest)
	if statusCode >= 400 {
		t.Fatalf("unable to deploy function: %d", statusCode)
	}

	return statusCode
}

func list(t *testing.T, expectedStatusCode int) {
	functions, err := config.Client.ListFunctions(context.Background(), defaultNamespace)
	if err != nil {
		t.Fatal(err)
	}

	if len(functions) == 0 {
		t.Fatal("List functions got: 0, want: > 0")
	}
}

func get(t *testing.T, name string) types.FunctionStatus {
	function, err := config.Client.GetFunctionInfo(context.Background(), name, defaultNamespace)
	if err != nil {
		t.Fatal(err)
	}

	return function
}

func deleteFunction(t *testing.T, function *sdk.DeployFunctionSpec) {
	t.Helper()

	err := config.Client.DeleteFunction(
		context.Background(),
		function.FunctionName,
		function.Namespace,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func scaleFunction(t *testing.T, name string, count int) {
	t.Helper()

	// the CLI sdk does not currently support manually scaling
	gwURL := resourceURL(t, path.Join("system", "scale-function", name), "")
	payload := makeReader(map[string]interface{}{"service": name, "replicas": count})

	// TODO : enable auth
	_, res := request(t, gwURL, http.MethodPost, config.Auth, payload)
	if res.StatusCode != http.StatusAccepted && res.StatusCode != http.StatusOK {
		t.Fatalf("scale got %d, wanted %d (or %d)", res.StatusCode, http.StatusAccepted, http.StatusOK)
	}
}
