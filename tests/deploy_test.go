package tests

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alexellis/faas/gateway/requests"
)

func Test_Pipeline(t *testing.T) {
	envVars := map[string]string{}
	deploy := requests.CreateFunctionRequest{
		Image:      "functions/alpine:latest",
		Service:    "stronghash",
		Network:    "func_functions",
		EnvProcess: "sha512sum",
		EnvVars:    envVars,
	}

	DeployTest(t, deploy)

	TestList(t)
}

func Test_PassingCustomEnvVars(t *testing.T) {
	envVars := map[string]string{}
	envVars["custom_env"] = "custom_env_value"

	deploy := requests.CreateFunctionRequest{
		Image:      "functions/alpine:latest",
		Service:    "env-test",
		Network:    "func_functions",
		EnvProcess: "env",
		EnvVars:    envVars,
	}
	DeployTest(t, deploy)
	TestList(t)
	AssertInvoke(t, deploy.Service, "custom_env")
}

func AssertInvoke(t *testing.T, name string, expected string) {
	attempts := 30 // i.e. 30x2s = 1m
	delay := time.Millisecond * 2000

	for i := 0; i < attempts; i++ {

		uri := os.Getenv("gateway_url") + "function/" + name

		bytesOut, res, err := httpReq(uri, "POST", nil)

		if err != nil {
			t.Log(err.Error())
			t.Fail()
		}

		if res.StatusCode != http.StatusOK {
			t.Logf("[%d/%d] Bad response want: %d, got: %d", i+1, attempts, http.StatusOK, res.StatusCode)
			t.Logf(uri)
			if i == attempts-1 {
				t.Logf("Failing after: %d attempts", attempts)
				t.Fail()
			}
			time.Sleep(delay)
			continue
		} else {
			t.Logf("[%d/%d] Correct response: %d", i+1, attempts, res.StatusCode)
		}

		out := string(bytesOut)
		if strings.Contains(out, expected) == false {
			t.Logf("want: %s, got: %s", expected, out)
			t.Fail()
		} else {
			break
		}
	}
}

func DeployTest(t *testing.T, createRequest requests.CreateFunctionRequest) {

	_, res, err := httpReq(os.Getenv("gateway_url")+"system/functions", "POST", makeReader(createRequest))
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	if res.StatusCode != http.StatusOK {
		t.Logf("got %d, wanted %d", res.StatusCode, http.StatusOK)
		t.Fail()
	}
}

func TestList(t *testing.T) {

	bytesOut, res, err := httpReq(os.Getenv("gateway_url")+"system/functions", "GET", nil)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	if res.StatusCode != http.StatusOK {
		t.Logf("got %d, wanted %d", res.StatusCode, http.StatusOK)
		t.Fail()
	}

	functions := []requests.Function{}
	err = json.Unmarshal(bytesOut, &functions)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	if len(functions) == 0 {
		t.Log("List functions got: 0, want: > 0")
		t.Fail()
	}
}
