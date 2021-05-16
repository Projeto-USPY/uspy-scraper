package icmc

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity"
	"github.com/Projeto-USPY/uspy-scraper/scraper"
	"github.com/Projeto-USPY/uspy-scraper/scraper/offerings"
	"github.com/PuerkitoBio/goquery"
)

var (
	DefaultICMCURLMask = "https://www.icmc.usp.br/templates/icmc2015/php/pessoas.php"
	ErrInvalidPage     = errors.New("last page reached")
)

type ICMCPeopleScraper struct {
	URLMask string
	Body    map[string]string
}

func NewICMCPeopleScraper(body map[string]string) ICMCPeopleScraper {
	return ICMCPeopleScraper{
		URLMask: DefaultICMCURLMask,
		Body:    body,
	}
}

func (os ICMCPeopleScraper) Start() (db.Writer, error) {
	data := url.Values{}
	for k, v := range os.Body {
		data.Set(k, v)
	}

	headers := map[string]string{
		"Content-Type":   "application/x-www-form-urlencoded",
		"Content-Length": strconv.Itoa(len(data.Encode())),
	}

	return scraper.Start(os, os.URLMask, http.MethodPost, strings.NewReader(data.Encode()), headers)
}

func (os ICMCPeopleScraper) Scrape(reader io.Reader) (obj db.Writer, err error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	var inst entity.Institute

	sel := doc.Find(".caption > a")

	for i := 0; i < sel.Length(); i++ {
		ithSel := sel.Eq(i)

		if href, ok := ithSel.Attr("href"); ok {
			seps := strings.Split(href, "=")
			if len(seps) > 1 {
				code, profName := seps[1], ithSel.Text()

				if num, err := strconv.Atoi(code); err != nil {
					return nil, err
				} else {
					codPes := strconv.Itoa((num - 3) / 2)
					uraniaSc := offerings.NewUraniaScraper(codPes, "2015", profName)
					result, err := uraniaSc.Start()

					if len(result.(entity.Professor).Offerings) == 0 {
						log.Println("found no offerings for ", codPes)
					}

					if err != nil {
						return nil, err
					}

					inst.Professors = append(inst.Professors, result.(entity.Professor))
				}

			}
		} else {
			return nil, errors.New("failed to fetch professor href")
		}
	}

	return inst, nil
}
