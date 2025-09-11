package adapters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type HTTPAdapter struct {
	client *http.Client
}

func NewHTTPAdapter() *HTTPAdapter {
	return &HTTPAdapter{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (a *HTTPAdapter) InvokeHTTP(endpoint, method string, input map[string]interface{}) (interface{}, error) {
	parts := strings.SplitN(method, " ", 2)
	httpMethod := "POST"
	path := "/"

	if len(parts) == 2 {
		httpMethod = parts[0]
		path = parts[1]
	} else if strings.HasPrefix(method, "/") {
		path = method
	} else {
		path = "/api/" + strings.ToLower(method)
	}

	url := endpoint + path

	var req *http.Request
	var err error

	if httpMethod == "GET" {
		req, err = http.NewRequest(httpMethod, url, nil)
		if err != nil {
			return nil, err
		}
		q := req.URL.Query()
		for k, v := range input {
			q.Add(k, fmt.Sprintf("%v", v))
		}
		req.URL.RawQuery = q.Encode()
	} else {
		body, err := json.Marshal(input)
		if err != nil {
			return nil, err
		}
		req, err = http.NewRequest(httpMethod, url, bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return string(body), nil
	}

	return result, nil
}
