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
	"github.com/rakyll/hey/requester"
)

func Test_ScaleMinimum(t *testing.T) {
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
	if *swarm {
		t.Skip("scale to zero currently returns 500 in faas-swarm")
	}
	functionName := "test-scale-from-zero"
	functionRequest := &sdk.DeployFunctionSpec{
		Image:        "functions/alpine:latest",
		FunctionName: functionName,
		Network:      "func_functions",
		FProcess:     "sha512sum",
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
	functionRequest := &sdk.DeployFunctionSpec{
		Image:        "functions/alpine:latest",
		FunctionName: functionName,
		Network:      "func_functions",
		FProcess:     "sha512sum",
		Labels:       labels,
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}

	defer deleteFunction(t, functionName)

	attempts := 1000
	output, cancelLoad := sendRequestLoad(t, functionName, attempts)
	replicas, cancelWatch := watchReplicaCount(t, functionName, maxReplicas)

	var loadOutput string
	var foundReplicas uint64
	select {
	case loadOutput = <-output:
		// end of load
		cancelLoad()
		cancelWatch()
		t.Log("request load finished")
	case foundReplicas = <-replicas:
		// reached desired replica count
		cancelLoad()
		cancelWatch()
	}

	if foundReplicas != maxReplicas {
		t.Logf("function load output %s", loadOutput)
		t.Fatalf("never reached max scale %d, only %d replicas after %d attempts", maxReplicas, foundReplicas, attempts)
	}

	replicas, cancelWatch = watchReplicaCount(t, functionName, minReplicas)
	defer cancelWatch()

	select {
	case <-replicas:
		return
	case <-time.After(65 * time.Second):
		fnc := get(t, functionName)
		if fnc.AvailableReplicas != minReplicas {
			t.Fatalf("got %d replicas, wanted %d", fnc.Replicas, minReplicas)
		}
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
	functionRequest := &sdk.DeployFunctionSpec{
		Image:        "functions/alpine:latest",
		FunctionName: functionName,
		Network:      "func_functions",
		FProcess:     "sha512sum",
		Labels:       labels,
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}

	defer deleteFunction(t, functionName)

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

	// don't use watchReplicaCount here because the function starts and should stay at the
	// same replica count throughout the entire load, here we want to just run through the entire
	// and check the replica count at the end
	fnc := get(t, functionName)
	if fnc.AvailableReplicas != minReplicas {
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
	functionRequest := &sdk.DeployFunctionSpec{
		Image:        "functions/alpine:latest",
		FunctionName: functionName,
		Network:      "func_functions",
		FProcess:     "sha512sum",
		Labels:       labels,
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}

	defer deleteFunction(t, functionName)

	attempts := 1000
	output, cancelLoad := sendRequestLoad(t, functionName, attempts)
	replicas, cancelWatch := watchReplicaCount(t, functionName, maxReplicas)

	var loadOutput string
	var foundReplicas uint64
	select {
	case loadOutput = <-output:
		// end of load
		cancelLoad()
		cancelWatch()
		t.Log("request load finished")
	case foundReplicas = <-replicas:
		// reached desired replica count
		cancelLoad()
		cancelWatch()
	}

	if foundReplicas != maxReplicas {
		t.Logf("function load output %s", loadOutput)
		t.Fatalf("never reached max scale %d, only %d replicas after %d attempts", maxReplicas, foundReplicas, attempts)
	}

	// cooldown
	replicas, cancelWatch = watchReplicaCount(t, functionName, 0)
	defer cancelWatch()

	select {
	case <-replicas:
		return
	case <-time.After(65 * time.Second):
		fnc := get(t, functionName)
		if fnc.AvailableReplicas != 0 {
			t.Fatalf("got %d replicas, wanted %d", fnc.Replicas, 0)
		}
	}
}

// sendRequestLoad is a helper method that will construct and run the hey Worker in a separate
// goroutine. When the worker finished, the output will be sent over the channel.  The caller
// can use the cancel method to stop the worker early
func sendRequestLoad(t *testing.T, functionName string, loadAttempts int) (_ <-chan string, cancel func()) {
	results := make(chan string)

	functionURL := resourceURL(t, path.Join("function", functionName), "")
	req, err := http.NewRequest(http.MethodPost, functionURL, nil)
	if err != nil {
		t.Fatalf("error building function load request %s ", err)
	}

	var loadOutput bytes.Buffer
	functionLoad := requester.Work{
		Request:           req,
		N:                 loadAttempts,
		Timeout:           10,
		C:                 2,
		QPS:               5.0,
		DisableKeepAlives: true,
		Writer:            &loadOutput,
	}

	go func() {
		functionLoad.Init()
		functionLoad.Run()

		results <- loadOutput.String()
		close(results)
	}()

	return results, functionLoad.Stop
}

// watchReplicaCount is helper method what will use the OpenFaaS API to watch and wait for a
// function to reach a desired available replica count. Once reached, the count is sent
// over the result channel. The caller can use the cancel method to stop the replica watcher early.
func watchReplicaCount(t *testing.T, functionName string, target uint64) (_ <-chan uint64, cancel func()) {
	t.Helper()
	results := make(chan uint64)

	ctx, cancel := context.WithCancel(context.Background())

	go func(t *testing.T) {
		defer close(results)
		start := time.Now()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				fnc := get(t, functionName)
				if fnc.AvailableReplicas == target {
					t.Logf("reached desired replicas '%d' in %s", target, time.Since(start))
					results <- fnc.AvailableReplicas
					return
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}(t)

	return results, cancel
}
