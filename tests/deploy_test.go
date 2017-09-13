package tests

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/alexellis/faas/gateway/requests"
)

func Test_Pipeline(t *testing.T) {
	TestDeploy(t)
	TestList(t)
}

func TestDeploy(t *testing.T) {
	deploy := requests.CreateFunctionRequest{
		Image:      "functions/alpine:latest",
		Service:    "stronghash",
		Network:    "func_functions",
		EnvProcess: "sha512sum",
	}

	_, res, err := post(os.Getenv("gateway_url")+"system/functions", "POST", makeReader(deploy))
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

	bytesOut, res, err := post(os.Getenv("gateway_url")+"system/functions", "GET", nil)
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
