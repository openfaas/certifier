package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"testing"
)

// gatewayURL safely constructs the API url based on the `gateway_url`
// in the ENV.
func gatewayURL(t *testing.T) string {
	t.Helper()
	uri, err := url.Parse(os.Getenv("gateway_url"))
	if err != nil {
		t.Fatalf("invalid gateway url %s", err)
	}
	return uri.String()
}

func resourceURL(t *testing.T, reqPath, query string) string {
	t.Helper()
	uri, err := url.Parse(os.Getenv("gateway_url"))
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

func request(t *testing.T, url, method string, reader io.Reader) ([]byte, *http.Response) {
	t.Helper()
	return requestContext(t, context.Background(), url, method, reader)
}

func requestContext(t *testing.T, ctx context.Context, url, method string, reader io.Reader) ([]byte, *http.Response) {
	t.Helper()

	c := http.Client{}

	req, makeReqErr := http.NewRequest(method, url, reader)
	if makeReqErr != nil {
		t.Fatalf("error with request %s ", makeReqErr)
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
