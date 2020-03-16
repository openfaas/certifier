package tests

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
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
	attempts := 1000
	functionLoad := requester.Work{
		Request:           req,
		N:                 attempts,
		Timeout:           10,
		C:                 50,
		QPS:               50.0,
		DisableKeepAlives: true,
		Writer:            &loadOutput,
	}

	functionLoad.Init()
	functionLoad.Run()

	fnc := get(t, functionName)
	if fnc.Replicas != maxReplicas {
		t.Logf("function load output %s", loadOutput.String())
		t.Fatalf("never reached max scale %d, only %d replicas after %d attempts", maxReplicas, fnc.Replicas, attempts)
	}

	// cooldown
	time.Sleep(time.Minute)
	fnc = get(t, functionName)
	if fnc.Replicas != minReplicas {
		t.Fatalf("got %d replicas, wanted %d", fnc.Replicas, minReplicas)
	}
}

func Test_ScalingDisabledViaLabels(t *testing.T) {
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
	attempts := 1000
	functionLoad := requester.Work{
		Request:           req,
		N:                 attempts,
		Timeout:           10,
		C:                 5,
		DisableKeepAlives: true,
		Writer:            &loadOutput,
	}

	functionLoad.Init()
	functionLoad.Run()

	fnc := get(t, functionName)
	if fnc.Replicas != minReplicas {
		t.Logf("function load output %s", loadOutput.String())
		t.Fatalf("unexpected scaling, expected %d, got %d replicas after %d attempts", minReplicas, fnc.Replicas, attempts)
	}
}

func Test_ScaleToZero(t *testing.T) {

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
	attempts := 1000
	functionLoad := requester.Work{
		Request:           req,
		N:                 attempts,
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
		t.Fatalf("never reached max scale %d, only %d replicas after %d attempts", maxReplicas, fnc.Replicas, attempts)
	}

	// cooldown
	time.Sleep(2 * time.Minute)
	fnc = get(t, functionName)
	if fnc.Replicas != 0 {
		t.Fatalf("got %d replicas, wanted 0", fnc.Replicas)
	}
}
