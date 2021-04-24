package tests

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	sdk "github.com/openfaas/faas-cli/proxy"
	"github.com/openfaas/faas-provider/logs"
)

func Test_FunctionLogs(t *testing.T) {
	type logsTestCase struct {
		name         string
		function     *sdk.DeployFunctionSpec
		expectedLogs []string
	}

	cases := []logsTestCase{
		{
			name: "provider can stream logs",
			function: &sdk.DeployFunctionSpec{
				Image:        "functions/alpine:latest",
				FunctionName: "test-logger",
				Network:      "func_functions",
				FProcess:     "sha512sum",
			},
			expectedLogs: []string{
				"Forking fprocess",
				"Wrote 132 Bytes",
			},
		},
	}

	if len(config.Namespaces) > 0 {
		cnCases := make([]logsTestCase, len(cases))
		copy(cnCases, cases)
		for index := 0; index < len(cnCases); index++ {
			cnCases[index].name = fmt.Sprintf("%s from %s", cnCases[index].name, config.Namespaces[0])
			cnCases[index].function.Namespace = config.Namespaces[0]
		}

		cases = append(cases, cnCases...)
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {

			deployStatus := deploy(t, c.function)
			if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
				t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
			}

			// each invoke should output two lines
			// - Forking fprocess.
			// - Wrote 132 Bytes - Duration: ...
			name := c.function.FunctionName
			if c.function.Namespace != "" {
				name = name + "." + c.function.Namespace
			}
			_ = invoke(t, name, "", http.StatusOK)

			logRequest := logs.Request{
				Name:      c.function.FunctionName,
				Namespace: c.function.Namespace,
				Tail:      2,
				Follow:    false,
			}

			// use context with timeout here to ensure we don't hang waiting for logs too long
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			logChan, err := config.Client.GetLogs(ctx, logRequest)
			if err != nil {
				t.Fatal(err)
			}

			logLines := []logs.Message{}
			for msg := range logChan {

				if msg.Name != c.function.FunctionName {
					t.Fatalf("got function name %s, expected %s", msg.Name, c.function.FunctionName)
				}

				if msg.Namespace != c.function.Namespace {
					t.Fatalf("got function namespace %s, expected %s", msg.Namespace, c.function.Namespace)
				}

				logLines = append(logLines, msg)
			}

			if len(logLines) != len(c.expectedLogs) {
				t.Fatalf("got %d lines, expected %d", len(logLines), len(c.expectedLogs))
			}

			for idx, expected := range c.expectedLogs {
				msg := logLines[idx]
				// remove the timstamp and white space prefix
				actual := strings.TrimLeft(msg.Text, "0123456789/: ")
				if !strings.HasPrefix(actual, expected) {
					t.Fatalf("got unexpected log message %q, expected %q", actual, expected)
				}
			}
		})
	}
}
