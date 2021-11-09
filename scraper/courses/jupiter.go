package courses

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity/models"
	"github.com/Projeto-USPY/uspy-scraper/scraper"
	"github.com/PuerkitoBio/goquery"
)

var (
	DefaultInstituteURLMask = "https://uspdigital.usp.br/jupiterweb/jupCursoLista?codcg=%s&tipo=N"
)

type JupiterScraper struct {
	URLMask string
	Code    string
}

func NewJupiterScraper(institute string) JupiterScraper {
	return JupiterScraper{
		URLMask: DefaultInstituteURLMask,
		Code:    institute,
	}
}

func (sc JupiterScraper) Start() (db.Writer, error) {
	URL := fmt.Sprintf(sc.URLMask, sc.Code)
	return scraper.Start(sc, URL, http.MethodGet, nil, nil, true)
}

func (sc JupiterScraper) Scrape(reader io.Reader) (obj db.Writer, err error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	institute := models.Institute{
		Name:    strings.TrimSpace(doc.Find("span > b").Text()),
		Code:    sc.Code,
		Courses: make([]models.Course, 0, 50),
	}

	coursesHref := doc.Find("td[valign=\"top\"] a.link_gray")
	for i := 0; i < coursesHref.Length(); i++ {
		// follow every course href
		node := coursesHref.Eq(i)
		if courseCode, courseSpec, err := getCourseIdentifiers(node); err != nil {
			return nil, err
		} else {
			courseScraper := NewCourseScraper(sc.Code, courseCode, courseSpec)
			if course, err := courseScraper.Start(); err != nil {
				return nil, err
			} else {
				institute.Courses = append(institute.Courses, course.(models.Course))
			}
		}
	}
	return institute, nil
}

func getCourseIdentifiers(node *goquery.Selection) (string, string, error) {
	if courseURL, exists := node.Attr("href"); exists {
		// get course code and specialization code
		regexCode := regexp.MustCompile(`codcur=(\d+)&codhab=(\d+)`)
		courseCodeMatches := regexCode.FindStringSubmatch(courseURL)

		if len(courseCodeMatches) < 3 {
			return "", "", ErrorCourseNotExist
		}

		courseCode, courseSpec := courseCodeMatches[1], courseCodeMatches[2]
		return courseCode, courseSpec, nil
	} else {
		return "", "", ErrorCourseNotExist
	}
}
