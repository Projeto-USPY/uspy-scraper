package scraper

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Projeto-USPY/uspy-scraper/utils"
	"github.com/PuerkitoBio/goquery"
)

const DefaultJupiterURLMask = "https://uspdigital.usp.br/jupiterweb/jupColegiadoLista?tipo=D"

func GetAllInstitutes() ([]string, error) {
	resp, reader, err := Fetch(DefaultJupiterURLMask, http.MethodGet, nil, nil, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	institutes := doc.Find(`td[valign="top"]:first-child`)
	var instituteCodes []string
	for i := 0; i < institutes.Length(); i++ {
		node := institutes.Eq(i)
		code := node.Text()
		cleanCode := strings.Trim(code, " \t\n")
		instituteCodes = append(instituteCodes, cleanCode)
	}

	return instituteCodes, nil
}

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
