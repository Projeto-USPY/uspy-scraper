package scraper

import (
	"fmt"
	"io"
	"net/http"

	"github.com/Projeto-USPY/uspy-scraper/utils"
)

func Fetch(startURL string, method string, body io.Reader, headers map[string]string, infereContentType bool) (*http.Response, io.Reader, error) {
	client := &http.Client{
		Timeout: 0,
	}

	resp, reader, err := utils.MakeRequestWithUTF8(client, startURL, method, body, headers, infereContentType)

	if err != nil {
		return nil, nil, err
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// successfully got start page
		return resp, reader, nil
	} else {
		return nil, nil, fmt.Errorf("could not fetch page: may be invalid url: %s, status is: %d", startURL, resp.StatusCode)
	}
}
