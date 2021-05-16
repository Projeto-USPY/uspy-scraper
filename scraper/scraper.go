package scraper

import (
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

func Start(scraper Scraper, startURL string, method string, body io.Reader, headers map[string]string) (db.Writer, error) {
	client := &http.Client{
		Timeout: 0,
	}

	var object db.Writer

	if resp, reader, err := utils.MakeRequestWithUTF8(client, startURL, method, body, headers); err != nil {
		return nil, err
	} else {
		// successfully got start page
		defer resp.Body.Close()
		if object, err = scraper.Scrape(reader); err != nil {
			return nil, err
		}
	}

	return object, nil
}
