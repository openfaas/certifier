package tests

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	faasSDK "github.com/openfaas/faas-cli/proxy"
	"github.com/openfaas/faas-provider/logs"
)

func Test_FunctionLogs(t *testing.T) {
	functionName := "test-logger"
	functionRequest := &faasSDK.DeployFunctionSpec{
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

	// each invoke should output two lines
	// - Forking fprocess.
	// - Wrote 132 Bytes - Duration: ...
	_ = invoke(t, functionName, "", http.StatusOK)

	logRequest := logs.Request{Name: functionName, Tail: 2, Follow: false}
	gwURL := gatewayUrl(t, "", "")

	// use context with timeout here to ensure we don't hang waiting for logs too long
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := faasSDK.NewClient(&FaaSAuth{}, gwURL, nil, &timeout)
	logChan, err := client.GetLogs(ctx, logRequest)
	if err != nil {
		t.Fatal(err)
	}

	// var b strings.Builder
	// for v := range logChan {
	// 	for _, line := range strings.Split(strings.TrimSuffix(v.Text, "\n"), "\n") {
	// 		b.WriteString(line + "\n")
	// 	}
	// }
	// logMessage := strings.TrimRight(b.String(), "\n")

	// stream := json.NewDecoder(bytes.NewReader([]byte(logMessage)))
	logLines := []logs.Message{}

	expectedTextA := "Forking fprocess"
	expectedTextB := "Wrote 132 Bytes"
	for msg := range logChan {

		if msg.Name != functionName {
			t.Fatalf("got function name %s, expected %s", msg.Name, functionName)
		}

		// remove the timstamp and white space prefix
		txt := strings.TrimLeft(msg.Text, "0123456789/: ")
		if !strings.HasPrefix(txt, expectedTextA) && !strings.HasPrefix(txt, expectedTextB) {
			t.Fatalf("got unexpected log message %q, expected %q or %q", txt, expectedTextA, expectedTextB)
		}

		logLines = append(logLines, msg)
	}

	if len(logLines) != 2 {
		t.Fatalf("got %d lines, expected %d", len(logLines), 2)
	}
}
