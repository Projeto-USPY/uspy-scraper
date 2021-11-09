package scraper

import (
	"fmt"
	"io"
	"net/http"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-scraper/utils"
)

type Starter interface {
	Start() (db.Writer, error)
}

type Scraper interface {
	Starter
	Scrape(reader io.Reader) (db.Writer, error)
}

func Start(scraper Scraper, startURL string, method string, body io.Reader, headers map[string]string, infereContentType bool) (db.Writer, error) {
	client := &http.Client{
		Timeout: 0,
	}

	var object db.Writer

	if resp, reader, err := utils.MakeRequestWithUTF8(client, startURL, method, body, headers, infereContentType); err != nil {
		return nil, err
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// successfully got start page
		defer resp.Body.Close()
		if object, err = scraper.Scrape(reader); err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("could not start scraper: invalid url:" + startURL)
	}

	return object, nil
}
