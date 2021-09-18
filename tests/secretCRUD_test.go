package tests

import (
	"context"
	"net/http"
	"testing"
	"time"

	sdk "github.com/openfaas/faas-cli/proxy"
	"github.com/openfaas/faas-provider/types"
)

func Test_SecretCRUD(t *testing.T) {
	setValue := "this-is-the-secret-value"
	setName := "secret-name"
	functionName := "test-secret-crud"

	ctx := context.Background()

	createStatus, _ := config.Client.CreateSecret(ctx, types.Secret{Name: setName, Value: setValue})
	if createStatus != http.StatusOK && createStatus != http.StatusAccepted {
		t.Fatalf("got %d, wanted %d or %d", createStatus, http.StatusOK, http.StatusAccepted)
	}
	t.Logf("Got correct response for creating secret: %d", createStatus)

	// Set up and deploy function that reads the value of the created secret.
	functionRequest := &sdk.DeployFunctionSpec{
		Image:        "functions/alpine:latest",
		FunctionName: functionName,
		Network:      "func_functions",
		FProcess:     "cat /var/openfaas/secrets/" + setName,
		Secrets:      []string{setName},
		Namespace:    config.DefaultNamespace,
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Errorf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}
	t.Logf("Got correct response for deploying function: %d", deployStatus)

	// Verify that the secret value was set as intended.
	value := string(invoke(t, functionRequest, "", "", http.StatusOK))
	if value != setValue {
		t.Errorf("got %s, wanted %s", value, setValue)
	}

	// Verify that the secret can be listed.
	secrets, err := config.Client.GetSecretList(ctx, config.DefaultNamespace)
	if err != nil {
		t.Fatal(err)
	}

	if !listContains(secrets, setName) {
		t.Errorf("got %v, wanted %s in slice", secrets, setName)
	}

	newValue := "this-is-the-edited-secret-value"
	updateStatus, _ := config.Client.UpdateSecret(ctx, types.Secret{Name: setName, Value: newValue})
	if updateStatus != http.StatusOK && updateStatus != http.StatusAccepted {
		t.Errorf("got %d, wanted %d or %d", updateStatus, http.StatusOK, http.StatusAccepted)
	}
	t.Logf("Got correct response for updating secret: %d", updateStatus)

	if config.ProviderName == faasNetesProviderName {
		err = config.Client.ScaleFunction(ctx, functionName, config.DefaultNamespace, 0)
		if err != nil {
			t.Error("Scaling down function to zero failed!")
		}
		t.Log("Scale Down function to zero")
		time.Sleep(time.Minute)
		err = config.Client.ScaleFunction(ctx, functionName, config.DefaultNamespace, 1)
		if err != nil {
			t.Error("Scaling up function from zero failed!")
		}
		t.Log("Scale up function from zero")
		time.Sleep(time.Minute)
	}

	// Verify that the secret value was edited.
	value = string(invoke(t, functionRequest, "", "", http.StatusOK))
	if value != newValue {
		t.Errorf("got %s, wanted %s", value, newValue)
	}

	// Function needs to be deleted to free up the secret so it can also be deleted.
	err = config.Client.DeleteFunction(ctx, functionRequest.FunctionName, functionRequest.Namespace)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Got correct response for deleting function")

	err = config.Client.RemoveSecret(ctx, types.Secret{Name: setName})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Got correct response for deleting secret:")

	// Verify that the secret was deleted.
	secrets, err = config.Client.GetSecretList(ctx, config.DefaultNamespace)
	if err != nil {
		t.Fatal(err)
	}
	if listContains(secrets, setName) {
		t.Errorf("got %v, wanted %s deleted", secrets, setName)
	}
}

func listContains(list []types.Secret, s string) bool {
	for i := range list {
		if list[i].Name == s {
			return true
		}
	}
	return false
}
