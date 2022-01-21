package icmc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Projeto-USPY/uspy-backend/entity/models"
	"github.com/Projeto-USPY/uspy-scraper/processor"
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

func (sc *ICMCPeopleScraper) Process(ctx context.Context) func(context.Context) (processor.Processed, error) {
	return func(context.Context) (processor.Processed, error) {
		data := url.Values{}
		for k, v := range sc.Body {
			data.Set(k, v)
		}

		headers := map[string]string{
			"Content-Type":   "application/x-www-form-urlencoded",
			"Content-Length": strconv.Itoa(len(data.Encode())),
		}

		resp, reader, err := scraper.Fetch(sc.URLMask, http.MethodPost, strings.NewReader(data.Encode()), headers, true)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromReader(reader)
		if err != nil {
			return nil, err
		}

		var inst models.Institute

		sel := doc.Find(".caption > a")

		offeringTasks := make([]*processor.Task, 0)

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
						offeringTasks = append(offeringTasks, processor.NewTask(
							fmt.Sprintf(
								"[offering-task] %s:%s",
								codPes,
								strings.ReplaceAll(strings.ToLower(profName), " ", "_"),
							),
							processor.QuadraticDelay,
							uraniaSc.Process(),
							nil,
						))
					}

				}
			} else {
				return nil, errors.New("failed to fetch professor href")
			}
		}

		proc := processor.NewProcessor(
			ctx,
			"[icmc-people-processor]",
			offeringTasks,
			processor.Config.FixedAttempts,
			processor.Config.DelayAttempts,
		)

		results := proc.Run()

		for _, result := range results {
			inst.Professors = append(inst.Professors, result.(models.Professor))
		}

		return inst, nil
	}
}
