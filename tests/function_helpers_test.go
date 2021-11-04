package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	sdk "github.com/openfaas/faas-cli/proxy"
	"github.com/openfaas/faas-cli/stack"
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
		t.Fatalf("unable to deploy function (%s.%s): %d",
			createRequest.FunctionName, createRequest.Namespace, statusCode)
	}

	return statusCode
}

func list(t *testing.T, expectedStatusCode int, namespace string) []types.FunctionStatus {
	functions, err := config.Client.ListFunctions(context.Background(), namespace)
	if err != nil {
		t.Fatal(err)
	}

	if len(functions) == 0 {
		t.Fatal("List functions got: 0, want: > 0")
	}

	return functions
}

func get(t *testing.T, name string, namespace string) types.FunctionStatus {
	function, err := config.Client.GetFunctionInfo(context.Background(), name, namespace)
	if err != nil {
		t.Fatalf("error getting info for: %s.%s, error: %s", name, namespace, err)
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
		t.Fatalf("unable to delete %s.%s, error: %s", function.FunctionName, function.Namespace, err)
	}
}

func copyNamespacesTest(cases []FunctionTestCase) []FunctionTestCase {
	// Add Test case, if CERTIFIER_NAMESPACES defined
	if len(config.Namespaces) > 0 {
		cnCases := make([]FunctionTestCase, len(cases))
		copy(cnCases, cases)
		for index := 0; index < len(cnCases); index++ {
			cnCases[index].name = fmt.Sprintf("%s to %s", cnCases[index].name, config.Namespaces[0])
			cnCases[index].function.Namespace = config.Namespaces[0]
		}

		cases = append(cases, cnCases...)
		return cases
	}
	return cases
}

func createDeploymentSpec(test FunctionTestCase) *sdk.DeployFunctionSpec {
	functionRequest := &sdk.DeployFunctionSpec{
		Image:        test.function.Image,
		FunctionName: test.function.Service,
		FProcess:     test.function.EnvProcess,
		EnvVars:      test.function.EnvVars,
		Namespace:    test.function.Namespace,
	}

	if test.function.Annotations != nil {
		functionRequest.Annotations = *test.function.Annotations
	}

	if test.function.Labels != nil {
		functionRequest.Labels = *test.function.Labels
	}

	if test.function.Limits != nil {
		limits := *test.function.Limits
		functionRequest.FunctionResourceRequest = sdk.FunctionResourceRequest{
			Limits: &stack.FunctionResources{
				Memory: limits.Memory,
				CPU:    limits.CPU,
			},
		}
	}

	return functionRequest
}

// waitForFunctionStatus polls the function status endpoint until the test function returns true _or_ the specified timeout.
func waitForFunctionStatus(timeout time.Duration, name, namespace string, test func(types.FunctionStatus) bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for ctx.Err() == nil {
		function, err := config.Client.GetFunctionInfo(ctx, name, namespace)
		if err != nil {
			return err
		}

		if test(function) {
			return nil
		}

		time.Sleep(time.Second)
	}

	return ctx.Err()
}

func maxReplicaCount(count uint64) func(types.FunctionStatus) bool {
	return func(function types.FunctionStatus) bool {
		return function.Replicas <= count
	}
}

func minReplicaCount(count uint64) func(types.FunctionStatus) bool {
	return func(function types.FunctionStatus) bool {
		return function.Replicas >= count
	}
}

func minAvailableReplicaCount(count uint64) func(types.FunctionStatus) bool {
	return func(function types.FunctionStatus) bool {
		return function.AvailableReplicas >= count
	}
}
