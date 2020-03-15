package tests

import (
	"bytes"
	"fmt"
	"net/http"
	"path"
	"testing"
	"time"

	"github.com/openfaas/faas/gateway/requests"

	"github.com/rakyll/hey/requester"
)

func Test_ScaleMinimum(t *testing.T) {
	functionName := "test-min-scale"
	minReplicas := uint64(2)
	labels := map[string]string{
		"com.openfaas.scale.min": fmt.Sprintf("%d", minReplicas),
	}
	functionRequest := requests.CreateFunctionRequest{
		Image:      "functions/alpine:latest",
		Service:    functionName,
		Network:    "func_functions",
		EnvProcess: "sha512sum",
		Labels:     &labels,
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}

	defer deleteFunction(t, functionName)

	fnc := get(t, functionName)
	if fnc.Replicas != minReplicas {
		t.Fatalf("got %d replicas, wanted %d", fnc.Replicas, minReplicas)
	}
}

func Test_ScaleFromZeroDuringInvoke(t *testing.T) {
	functionName := "test-scale-from-zero"
	functionRequest := requests.CreateFunctionRequest{
		Image:      "functions/alpine:latest",
		Service:    functionName,
		Network:    "func_functions",
		EnvProcess: "sha512sum",
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}

	defer deleteFunction(t, functionName)

	scaleFunction(t, functionName, 0)

	fnc := get(t, functionName)
	if fnc.Replicas != 0 {
		t.Fatalf("got %d replicas, wanted %d", fnc.Replicas, 0)
	}

	// this will fail or pass the test
	_ = invoke(t, functionName, "", http.StatusOK)
}

func Test_ScaleUpAndDownFromThroughPut(t *testing.T) {
	functionName := "test-throughput-scaling"
	minReplicas := uint64(1)
	maxReplicas := uint64(2)
	labels := map[string]string{
		"com.openfaas.scale.min": fmt.Sprintf("%d", minReplicas),
		"com.openfaas.scale.max": fmt.Sprintf("%d", maxReplicas),
	}
	functionRequest := requests.CreateFunctionRequest{
		Image:      "functions/alpine:latest",
		Service:    functionName,
		Network:    "func_functions",
		EnvProcess: "sha512sum",
		Labels:     &labels,
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}

	defer deleteFunction(t, functionName)

	functionURL := gatewayUrl(t, path.Join("function", functionName), "")
	req, err := http.NewRequest(http.MethodPost, functionURL, nil)
	if err != nil {
		t.Fatalf("error with request %s ", err)
	}

	var loadOutput bytes.Buffer
	functionLoad := requester.Work{
		Request:           req,
		N:                 1000,
		Timeout:           10,
		C:                 50,
		DisableKeepAlives: true,
		Writer:            &loadOutput,
	}

	functionLoad.Init()
	functionLoad.Run()

	fnc := get(t, functionName)
	if fnc.Replicas != maxReplicas {
		t.Logf("function load output %s", loadOutput.String())
		t.Fatalf("never reached max scale %d, only %d replicas after %d attempts", maxReplicas, fnc.Replicas, 1000)
	}

	// cooldown
	time.Sleep(time.Minute)
	fnc = get(t, functionName)
	if fnc.Replicas != minReplicas {
		t.Fatalf("got %d replicas, wanted %d", fnc.Replicas, minReplicas)
	}
}
