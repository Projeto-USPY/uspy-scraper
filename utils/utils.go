// package utils contain useful helper functions
package utils

import (
	"io"
	"net/http"

	"golang.org/x/net/html/charset"
)

// Returns HTTP response and io.Reader from http.Get, which should substitute http.Body, so characters are read with UTF-8 encoding
// Already panics if error, remember to close response.Body
func HTTPGetWithUTF8(client *http.Client, url string) (*http.Response, io.Reader, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, nil, err
	}

	reader, err := charset.NewReader(resp.Body, resp.Header["Content-Type"][0])
	if err != nil {
		return nil, nil, err
	}

	return resp, reader, nil
}
