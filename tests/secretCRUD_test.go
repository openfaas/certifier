package tests

import (
	"context"
	"net/http"
	"testing"

	sdk "github.com/openfaas/faas-cli/proxy"
	"github.com/openfaas/faas-provider/types"
)

func Test_SecretCRUD(t *testing.T) {
	setValue := "this-is-the-secret-value"
	setName := "secret-name"
	functionName := "test-secret-crud"

	ctx := context.Background()

	createStatus, _ := config.Client.CreateSecret(ctx, types.Secret{Name: setName, Value: setValue})
	if createStatus != http.StatusCreated && createStatus != http.StatusAccepted {
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
	}

	deployStatus := deploy(t, functionRequest)
	if deployStatus != http.StatusOK && deployStatus != http.StatusAccepted {
		t.Errorf("got %d, wanted %d or %d", deployStatus, http.StatusOK, http.StatusAccepted)
	}
	t.Logf("Got correct response for deploying function: %d", deployStatus)

	// Verify that the secret value was set as intended.
	value := string(invoke(t, functionRequest.FunctionName, "", "", http.StatusOK))
	if value != setValue {
		t.Errorf("got %s, wanted %s", value, setValue)
	}

	// Verify that the secret can be listed.
	secrets, err := config.Client.GetSecretList(ctx, defaultNamespace)
	if err != nil {
		t.Fatal(err)
	}

	if !listContains(secrets, setName) {
		t.Errorf("got %v, wanted %s in slice", secrets, setName)
	}

	// Docker Swarm secrets are immutable, so skip the update tests for swarm.
	if config.SecretUpdate {
		newValue := "this-is-the-edited-secret-value"
		updateStatus, _ := config.Client.UpdateSecret(ctx, types.Secret{Name: setName, Value: newValue})
		if updateStatus != http.StatusOK && updateStatus != http.StatusAccepted {
			t.Errorf("got %d, wanted %d or %d", updateStatus, http.StatusOK, http.StatusAccepted)
		}
		t.Logf("Got correct response for updating secret: %d", updateStatus)

		// Verify that the secret value was edited.
		value = string(invoke(t, functionRequest.FunctionName, "", "", http.StatusOK))
		if value != setValue {
			t.Errorf("got %s, wanted %s", value, newValue)
		}
	} else {
		t.Log("secret update skipped")
	}

	// Function needs to be deleted to free up the secret so it can also be deleted.
	err = config.Client.DeleteFunction(ctx, functionName, defaultNamespace)
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
	secrets, err = config.Client.GetSecretList(ctx, defaultNamespace)
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
