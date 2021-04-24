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
			cnCases[index].function.Namespace = ns
			cnCases[index].expectedLogs[1] = fmt.Sprintf("Wrote %d Bytes", len(ns))
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
			name := c.function.FunctionName
			if c.function.Namespace != "" {
				name = name + "." + c.function.Namespace
			}

			ns := c.function.Namespace
			if ns == "" {
				ns = config.DefaultNamespace
			}

			data := invoke(t, name, "", ns, http.StatusOK)
			if string(data) != ns {
				t.Fatalf("got invoke response %s, expected %s", string(data), ns)
			}

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
					t.Logf("got function namespace %s, expected %s", msg.Namespace, c.function.Namespace)
				}

				logLines = append(logLines, msg)
			}

			if len(logLines) != len(c.expectedLogs) {
				debug := strings.Builder{}
				for _, line := range logLines {
					debug.WriteString(line.Text)
				}
				t.Logf("recieved:\n%s\n", debug.String())
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
