// package utils contain useful helper functions
package utils

import (
	"bufio"
	"io"
	"net/http"

	"golang.org/x/net/html/charset"
)

var defaultHeaders = map[string]string{
	"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/80.0.3987.149 Safari/537.36",
}

// Returns HTTP response and io.Reader from http Request, which should substitute http.Body, so characters are read with UTF-8 encoding
// Remember to close response.Body
func MakeRequestWithUTF8(client *http.Client, url, method string, body io.Reader, headers map[string]string, infereContentType bool) (*http.Response, io.Reader, error) {
	req, err := http.NewRequest(method, url, body)

	for k, v := range defaultHeaders {
		req.Header.Add(k, v)
	}

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

	if infereContentType {
		reader, err := charset.NewReader(resp.Body, resp.Header.Get("Content-Type"))
		if err != nil {
			return nil, nil, err
		}

		return resp, reader, nil
	}

	return resp, bufio.NewReader(resp.Body), nil
}
