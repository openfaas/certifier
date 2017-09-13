package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

func makeReader(input interface{}) *bytes.Buffer {
	res, _ := json.Marshal(input)
	return bytes.NewBuffer(res)
}

func post(url1, method string, reader io.Reader) ([]byte, *http.Response, error) {
	c := http.Client{}

	req, makeReqErr := http.NewRequest(method, url1, reader)
	if makeReqErr != nil {
		return nil, nil, fmt.Errorf("error with request %s ", makeReqErr)
	}

	res, callErr := c.Do(req)
	if callErr != nil {
		return nil, nil, fmt.Errorf("call error %s ", callErr)
	}
	if res.Body != nil {
		defer res.Body.Close()
		bytesOut, err := ioutil.ReadAll(res.Body)

		return bytesOut, res, err
	}

	return nil, res, nil
}
