// package utils contain useful helper functions
package utils

import (
	"io"
	"net/http"

	"golang.org/x/net/html/charset"
)

// Returns HTTP response and io.Reader from http Request, which should substitute http.Body, so characters are read with UTF-8 encoding
// Remember to close response.Body
func MakeRequestWithUTF8(client *http.Client, url, method string, body io.Reader, headers map[string]string) (*http.Response, io.Reader, error) {
	req, err := http.NewRequest(method, url, body)
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	if err != nil {
		return nil, nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	reader, err := charset.NewReader(resp.Body, resp.Header["Content-Type"][0])
	if err != nil {
		return nil, nil, err
	}

	return resp, reader, nil
}
