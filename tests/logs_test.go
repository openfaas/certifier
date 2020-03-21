package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/openfaas/faas-provider/logs"
	"github.com/openfaas/faas-provider/types"
)

func Test_FunctionLogs(t *testing.T) {
	functionName := "test-logger"
	functionRequest := types.FunctionDeployment{
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

	// each invoke should output two lines
	// - Forking fprocess.
	// - Wrote 132 Bytes - Duration: ...
	_ = invoke(t, functionName, "", http.StatusOK)

	query := fmt.Sprintf("name=%s&tail=%d&follow=false", functionName, 2)
	gwURL := gatewayUrl(t, path.Join("system", "logs"), query)

	// use context with timeout here to ensure we don't hang waiting for logs too long
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logJSON, resp := requestContext(t, ctx, gwURL, http.MethodGet, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status code %d, expected 200", resp.StatusCode)
	}

	stream := json.NewDecoder(bytes.NewReader(logJSON))
	logLines := []logs.Message{}

	expectedTextA := "Forking fprocess"
	expectedTextB := "Wrote 132 Bytes"
	for stream.More() {
		msg := logs.Message{}
		err := stream.Decode(&msg)
		if err != nil {
			t.Fatalf("failed to parse log message %s", err)
		}

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
