package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	types "github.com/openfaas/faas-provider/types"
)

//ScaleFunction scale a function
func (c *Client) ScaleFunction(ctx context.Context, functionName, namespace string, replicas uint64) error {

	scaleReq := types.ScaleServiceRequest{
		ServiceName: functionName,
		Replicas:    replicas,
	}

	var err error

	bodyBytes, _ := json.Marshal(scaleReq)
	bodyReader := bytes.NewReader(bodyBytes)

	functionPath := filepath.Join(scalePath, functionName)
	if len(namespace) > 0 {
		functionPath, err = addQueryParams(functionPath, map[string]string{namespaceKey: namespace})
		if err != nil {
			return err
		}
	}

	req, err := c.newRequest(http.MethodPost, functionPath, bodyReader)
	if err != nil {
		return fmt.Errorf("cannot connect to OpenFaaS on URL: %s", c.GatewayURL.String())
	}

	res, err := c.doRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("cannot connect to OpenFaaS on URL: %s", c.GatewayURL.String())

	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	switch res.StatusCode {
	case http.StatusAccepted, http.StatusOK, http.StatusCreated:
		break

	case http.StatusNotFound:
		return fmt.Errorf("function %s not found", functionName)

	case http.StatusUnauthorized:
		return fmt.Errorf("unauthorized action, please setup authentication for this server")

	default:
		var bodyReadErr error
		bytesOut, bodyReadErr := ioutil.ReadAll(res.Body)
		if bodyReadErr != nil {
			return bodyReadErr
		}

		return fmt.Errorf("server returned unexpected status code %d %s", res.StatusCode, string(bytesOut))
	}
	return nil
}
