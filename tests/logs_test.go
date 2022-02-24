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
		function     sdk.DeployFunctionSpec
		expectedLogs []string
	}

	cases := []logsTestCase{
		{
			name: "provider can stream logs",
			function: sdk.DeployFunctionSpec{
				Image:        "functions/alpine:latest",
				FunctionName: "test-logger",
				Network:      "func_functions",
				FProcess:     "cat",
				Namespace:    config.DefaultNamespace,
			},
			expectedLogs: []string{
				"Forking fprocess",
				fmt.Sprintf("Wrote %d Bytes", len(config.DefaultNamespace)),
			},
		},
	}

	if len(config.Namespaces) > 0 {
		cnCases := make([]logsTestCase, len(cases))
		copy(cnCases, cases)
		for index := 0; index < len(cnCases); index++ {
			ns := config.Namespaces[0]

			newLogs := make([]string, len(cnCases[index].expectedLogs))
			copy(newLogs, cnCases[index].expectedLogs)

			newLogs[1] = fmt.Sprintf("Wrote %d Bytes", len(ns))
			cnCases[index].expectedLogs = newLogs
			cnCases[index].function.Namespace = ns
		}

		cases = append(cases, cnCases...)
	}

	for idx, c := range cases {
		// prefix the name with the index to avoid any possible mistakes that cause
		// duplicate cases
		t.Run(fmt.Sprintf("%d %s from %s", idx, c.name, c.function.Namespace), func(t *testing.T) {

			deployStatus := deploy(t, &c.function)
			if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
				t.Fatalf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
			}

			// each invoke should output two lines
			// - Forking fprocess.
			// - Wrote 132 Bytes - Duration: ...

			ns := c.function.Namespace
			if ns == "" {
				ns = config.DefaultNamespace
			}

			err := waitForFunctionStatus(time.Minute, c.function.FunctionName, ns, minAvailableReplicaCount(1))
			if err != nil {
				t.Fatalf("Function %q failed to start: %s", c.function.FunctionName, err)
			}

			data := invoke(t, &c.function, "", ns, http.StatusOK)
			if string(data) != ns {
				t.Fatalf("got invoke response %s, expected %s", string(data), ns)
			}

			time.Sleep(30 * time.Second)

			logRequest := logs.Request{
				Name:      c.function.FunctionName,
				Namespace: c.function.Namespace,
				Follow:    false,
			}

			// use context with timeout here to ensure we don't hang waiting for logs too long
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			logChan, err := config.Client.GetLogs(ctx, logRequest)
			if err != nil {
				t.Fatal(err)
			}

			logLines := []logs.Message{}
			for msg := range logChan {

				if msg.Name != c.function.FunctionName {
					t.Fatalf("function name got %s, want %s", msg.Name, c.function.FunctionName)
				}

				if msg.Namespace != c.function.Namespace {
					t.Logf("function got: %s, want: %s", msg.Namespace, c.function.Namespace)
				}

				logLines = append(logLines, msg)
			}

			for _, want := range c.expectedLogs {
				if !checkIfLogIsRecorded(logLines, want) {
					t.Fatalf("Want log message %q, but were not recorded", want)
				}
			}
		})
	}
}

func checkIfLogIsRecorded(logLines []logs.Message, expected string) bool {
	for _, msg := range logLines {
		actual := strings.TrimLeft(msg.Text, "0123456789/: ")
		if strings.HasPrefix(actual, expected) {
			return true
		}
	}
	return false
}
