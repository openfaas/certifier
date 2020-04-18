package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"testing"

	sdk "github.com/openfaas/faas-cli/proxy"
)

// resourceURL safely constructs the API url based on the `gateway_url`
// in the ENV.
func resourceURL(t *testing.T, reqPath, query string) string {
	t.Helper()
	uri, err := url.Parse(config.Gateway)
	if err != nil {
		t.Fatalf("invalid gateway url %s", err)
	}

	uri.Path = path.Join(uri.Path, reqPath)
	uri.RawQuery = query
	return uri.String()
}

func makeReader(input interface{}) *bytes.Buffer {
	res, _ := json.Marshal(input)
	return bytes.NewBuffer(res)
}

func request(t *testing.T, url, method string, auth sdk.ClientAuth, reader io.Reader) ([]byte, *http.Response) {
	t.Helper()
	return requestContext(t, context.Background(), url, method, auth, reader)
}

func requestContext(t *testing.T, ctx context.Context, url, method string, auth sdk.ClientAuth, reader io.Reader) ([]byte, *http.Response) {
	t.Helper()

	c := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, makeReqErr := http.NewRequest(method, url, reader)
	if makeReqErr != nil {
		t.Fatalf("error with request %s ", makeReqErr)
	}

	if auth != nil {
		err := auth.Set(req)
		if err != nil {
			t.Fatalf("error setting the request auth %s ", err)
		}
	}

	req = req.WithContext(ctx)

	res, callErr := c.Do(req)
	if callErr != nil {
		t.Fatalf("call error %s ", callErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
		bytesOut, err := ioutil.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("error reading response body %s ", err)
		}

		return bytesOut, res
	}

	return nil, res
}
