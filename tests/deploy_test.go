package tests

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/openfaas/faas/gateway/requests"
)

func Test_Deploy_Stronghash(t *testing.T) {
	envVars := map[string]string{}
	deploy := requests.CreateFunctionRequest{
		Image:      "functions/alpine:latest",
		Service:    "stronghash",
		Network:    "func_functions",
		EnvProcess: "sha512sum",
		EnvVars:    envVars,
	}

	deployStatus, deployErr := Deploy(t, deploy)
	if deployErr != nil {
		t.Log(deployErr.Error())
		t.Fail()
	}
	if deployStatus != http.StatusOK {
		t.Logf("got %d, wanted %d", deployStatus, http.StatusOK)
		t.Fail()
	}
	List(t, http.StatusOK)
}

func Test_InvokeNotFound(t *testing.T) {
	Invoke(t, "notfound", http.StatusNotFound)
}

func Test_Deploy_PassingCustomEnvVars(t *testing.T) {
	envVars := map[string]string{}
	envVars["custom_env"] = "custom_env_value"

	deploy := requests.CreateFunctionRequest{
		Image:      "functions/alpine:latest",
		Service:    "env-test",
		Network:    "func_functions",
		EnvProcess: "env",
		EnvVars:    envVars,
	}

	deployStatus, deployErr := Deploy(t, deploy)
	if deployErr != nil {
		t.Log(deployErr.Error())
		t.Fail()
	}
	if deployStatus != http.StatusOK {
		t.Logf("got %d, wanted %d", deployStatus, http.StatusOK)
		t.Fail()
	}

	List(t, http.StatusOK)

	bytesOut := Invoke(t, deploy.Service, http.StatusOK)

	out := string(bytesOut)
	if strings.Contains(out, "custom_env") == false {
		t.Logf("want: %s, got: %s", "custom_env", out)
		t.Fail()
	}
}

func Invoke(t *testing.T, name string, expectedStatusCode int) []byte {
	attempts := 30 // i.e. 30x2s = 1m
	delay := time.Millisecond * 2000

	for i := 0; i < attempts; i++ {

		uri := os.Getenv("gateway_url") + "function/" + name

		bytesOut, res, err := httpReq(uri, "POST", nil)

		if err != nil {
			t.Log(err.Error())
			t.Fail()
		}

		if res.StatusCode != expectedStatusCode {
			t.Logf("[%d/%d] Bad response want: %d, got: %d", i+1, attempts, expectedStatusCode, res.StatusCode)
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

		return bytesOut
	}
	return nil
}

func Deploy(t *testing.T, createRequest requests.CreateFunctionRequest) (int, error) {

	_, res, err := httpReq(os.Getenv("gateway_url")+"system/functions", "POST", makeReader(createRequest))
	if err != nil {
		return http.StatusBadGateway, err
	}

	return res.StatusCode, nil
}

func List(t *testing.T, expectedStatusCode int) {

	bytesOut, res, err := httpReq(os.Getenv("gateway_url")+"system/functions", "GET", nil)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	if res.StatusCode != expectedStatusCode {
		t.Logf("got %d, wanted %d", res.StatusCode, expectedStatusCode)
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
