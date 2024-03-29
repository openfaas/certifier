package tests

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	sdk "github.com/openfaas/faas-cli/proxy"
	types "github.com/openfaas/faas-provider/types"
	"github.com/rakyll/hey/requester"
)

func Test_ScaleMinimum(t *testing.T) {
	if !config.EnableScaling {
		t.Skipf("scale to minimum is not supported for %s", config.ProviderName)
	}
	functionName := "test-min-scale"
	minReplicas := uint64(2)
	labels := map[string]string{
		"com.openfaas.scale.min": fmt.Sprintf("%d", minReplicas),
	}
	functionRequest := &sdk.DeployFunctionSpec{
		Image:        "functions/alpine:latest",
		FunctionName: functionName,
		Network:      "func_functions",
		FProcess:     "sha512sum",
		Labels:       labels,
		Namespace:    config.DefaultNamespace,
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}

	defer deleteFunction(t, functionRequest)

	fnc := get(t, functionName, config.DefaultNamespace)
	if fnc.Replicas != minReplicas {
		t.Fatalf("got %d replicas, wanted %d", fnc.Replicas, minReplicas)
	}
}

func Test_ScaleFromZeroDuringInvoke(t *testing.T) {
	if !config.EnableScaling {
		t.Skipf("scale to zero is not supported for %s", config.ProviderName)
	}
	functionName := "test-scale-from-zero"
	functionRequest := &sdk.DeployFunctionSpec{
		Image:        "functions/alpine:latest",
		FunctionName: functionName,
		Network:      "func_functions",
		FProcess:     "sha512sum",
		Namespace:    config.DefaultNamespace,
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}
	defer deleteFunction(t, functionRequest)

	err := waitForFunctionStatus(time.Minute, functionName, config.DefaultNamespace, minAvailableReplicaCount(1))
	if err != nil {
		t.Fatalf("Function %q failed to start: %s", functionName, err)
	}

	err = config.Client.ScaleFunction(context.Background(), functionName, config.DefaultNamespace, 0)
	if err != nil {
		t.Fatalf("Scaling down function to zero failed: %s", err)
	}

	fnc := get(t, functionName, config.DefaultNamespace)
	if fnc.Replicas != 0 {
		t.Fatalf("got %d replicas, wanted %d", fnc.Replicas, 0)
	}

	// this will fail or pass the test
	_ = invoke(t, functionRequest, "", "", http.StatusOK)
}

func Test_ScaleUpAndDownFromThroughPut(t *testing.T) {
	if !config.EnableScaling {
		t.Skipf("scale up and down is not supported for %s", config.ProviderName)
	}
	functionName := "test-throughput-scaling"
	minReplicas := uint64(1)
	maxReplicas := uint64(2)
	labels := map[string]string{
		"com.openfaas.scale.min": fmt.Sprintf("%d", minReplicas),
		"com.openfaas.scale.max": fmt.Sprintf("%d", maxReplicas),
	}
	functionRequest := &sdk.DeployFunctionSpec{
		Image:        "functions/alpine:latest",
		FunctionName: functionName,
		Network:      "func_functions",
		FProcess:     "sha512sum",
		Labels:       labels,
		Namespace:    config.DefaultNamespace,
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}

	defer deleteFunction(t, functionRequest)

	functionURL := resourceURL(t, path.Join("function", functionName), "")
	req, err := http.NewRequest(http.MethodPost, functionURL, nil)
	if err != nil {
		t.Fatalf("error with request %s ", err)
	}

	var loadOutput bytes.Buffer
	attempts := 1000
	functionLoad := requester.Work{
		Request:           req,
		N:                 attempts,
		Timeout:           10,
		C:                 2,
		QPS:               5.0,
		DisableKeepAlives: true,
		Writer:            &loadOutput,
	}

	functionLoad.Init()
	go func() {
		functionLoad.Run()
	}()

	var status types.FunctionStatus
	_ = waitForFunctionStatus(time.Minute, functionName, config.DefaultNamespace, func(fnc types.FunctionStatus) bool {
		status = fnc
		if fnc.Replicas >= maxReplicas {
			functionLoad.Stop()
			return true
		}
		return false
	})

	if status.Replicas != maxReplicas {
		t.Logf("function load output %s", loadOutput.String())
		t.Fatalf("never reached max scale %d, only %d replicas after %d attempts", maxReplicas, status.Replicas, attempts)
	}

	// cooldown
	_ = waitForFunctionStatus(time.Minute, functionName, config.DefaultNamespace, maxReplicaCount(minReplicas))
	fnc := get(t, functionName, config.DefaultNamespace)
	if fnc.Replicas != minReplicas {
		t.Fatalf("got %d replicas, wanted %d", fnc.Replicas, minReplicas)
	}
}

func Test_ScalingDisabledViaLabels(t *testing.T) {
	if !config.EnableScaling {
		t.Skipf("scaling disabled via label is not supported for %s", config.ProviderName)
	}
	functionName := "test-scaling-disabled"
	minReplicas := uint64(2)
	maxReplicas := minReplicas
	// Per the docs, setting these values equal to each other will disabled
	// scaling
	// https://docs.openfaas.com/architecture/autoscaling/#minmax-replicas
	labels := map[string]string{
		"com.openfaas.scale.min": fmt.Sprintf("%d", minReplicas),
		"com.openfaas.scale.max": fmt.Sprintf("%d", maxReplicas),
	}
	functionRequest := &sdk.DeployFunctionSpec{
		Image:        "functions/alpine:latest",
		FunctionName: functionName,
		Network:      "func_functions",
		FProcess:     "sha512sum",
		Labels:       labels,
		Namespace:    config.DefaultNamespace,
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}

	defer deleteFunction(t, functionRequest)

	functionURL := resourceURL(t, path.Join("function", functionName), "")
	req, err := http.NewRequest(http.MethodPost, functionURL, nil)
	if err != nil {
		t.Fatalf("error with request %s ", err)
	}

	var loadOutput bytes.Buffer
	attempts := 1000
	functionLoad := requester.Work{
		Request:           req,
		N:                 attempts,
		Timeout:           10,
		C:                 2,
		QPS:               5.0,
		DisableKeepAlives: true,
		Writer:            &loadOutput,
	}

	functionLoad.Init()
	functionLoad.Run()

	fnc := get(t, functionName, config.DefaultNamespace)
	if fnc.Replicas != minReplicas {
		t.Logf("function load output %s", loadOutput.String())
		t.Fatalf("unexpected scaling, expected %d, got %d replicas after %d attempts", minReplicas, fnc.Replicas, attempts)
	}
}

func Test_ScaleToZero(t *testing.T) {
	if !config.EnableScaling {
		t.Skipf("scale to zero is not supported for %s", config.ProviderName)
	}

	idlerEnabled := os.Getenv("idler_enabled")
	if idlerEnabled == "" {
		idlerEnabled = "false"
	}

	enableTest, err := strconv.ParseBool(idlerEnabled)
	if err != nil {
		t.Fatal(err)
	}

	if !enableTest {
		t.Skip("set 'idler_enabled' to test scale to zero")
	}

	functionName := "test-scaling-to-zero"
	maxReplicas := uint64(2)
	labels := map[string]string{
		"com.openfaas.scale.max":  fmt.Sprintf("%d", maxReplicas),
		"com.openfaas.scale.zero": "true",
	}
	functionRequest := &sdk.DeployFunctionSpec{
		Image:        "functions/alpine:latest",
		FunctionName: functionName,
		Network:      "func_functions",
		FProcess:     "sha512sum",
		Labels:       labels,
		Namespace:    config.DefaultNamespace,
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}

	defer deleteFunction(t, functionRequest)

	functionURL := resourceURL(t, path.Join("function", functionName), "")
	req, err := http.NewRequest(http.MethodPost, functionURL, nil)
	if err != nil {
		t.Fatalf("error with request %s ", err)
	}

	var loadOutput bytes.Buffer
	attempts := 1000
	functionLoad := requester.Work{
		Request:           req,
		N:                 attempts,
		Timeout:           10,
		C:                 2,
		QPS:               5.0,
		DisableKeepAlives: true,
		Writer:            &loadOutput,
	}

	functionLoad.Init()
	functionLoad.Run()

	fnc := get(t, functionName, config.DefaultNamespace)
	if fnc.Replicas != maxReplicas {
		t.Logf("function load output %s", loadOutput.String())
		t.Fatalf("never reached max scale %d, only %d replicas after %d attempts", maxReplicas, fnc.Replicas, attempts)
	}

	// cooldown
	_ = waitForFunctionStatus(2*time.Minute, functionName, config.DefaultNamespace, minReplicaCount(0))
	fnc = get(t, functionName, config.DefaultNamespace)
	if fnc.Replicas != 0 {
		t.Fatalf("got %d replicas, wanted 0", fnc.Replicas)
	}
}
