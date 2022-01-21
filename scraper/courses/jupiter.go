package courses

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/Projeto-USPY/uspy-backend/entity/models"
	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper"
	"github.com/PuerkitoBio/goquery"
	log "github.com/sirupsen/logrus"
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

func (sc *JupiterScraper) Process(ctx context.Context) func(context.Context) (processor.Processed, error) {
	return func(context.Context) (processor.Processed, error) {
		URL := fmt.Sprintf(sc.URLMask, sc.Code)
		resp, reader, err := scraper.Fetch(URL, http.MethodGet, nil, nil, true)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromReader(reader)
		if err != nil {
			return nil, err
		}

		institute := models.Institute{
			Name:    strings.TrimSpace(doc.Find("span > b").Text()),
			Code:    sc.Code,
			Courses: make([]models.Course, 0, 50),
		}

		courseTasks := make([]*processor.Task, 0)

		courses := doc.Find(`td[valign="top"] > font > span`)
		if courses.Length()%2 != 0 { // odd is unexpected because each course should have a shift
			log.WithFields(log.Fields{
				"institute": sc.Code,
			}).Errorf("can't scrape institute, for some reason there's not a shift for each course")
			return nil, errors.New("can't scrape institute, for some reason there's not a shift for each course")
		}

		for i := 0; i < courses.Length(); i += 2 {
			// follow every course href
			courseNode := courses.Eq(i).Find("a.link_gray")
			shift := courses.Eq(i + 1).Text()
			cleanShift := strings.Trim(shift, " \n\t")

			if len(cleanShift) == 0 {
				log.WithFields(log.Fields{
					"institute": sc.Code,
				}).Errorf("can't scrape institute, for some reason there's a couse with an empty shift")
				return nil, errors.New("course shift is empty")
			}

			if courseCode, courseSpec, err := getCourseIdentifiers(courseNode); err != nil {
				return nil, err
			} else {
				courseScraper := NewCourseScraper(sc.Code, courseCode, courseSpec, cleanShift)
				courseTasks = append(courseTasks, processor.NewTask(
					fmt.Sprintf(
						"[course-task] %s:%s:%s",
						institute.Code,
						courseCode,
						cleanShift,
					),
					processor.QuadraticDelay,
					courseScraper.Process(),
					nil,
				))
			}
		}

		proc := processor.NewProcessor(
			ctx,
			fmt.Sprintf(
				"[institute-processor] %s",
				institute.Code,
			),
			courseTasks,
			true,
		)

		results := proc.Run()

		for _, course := range results {
			institute.Courses = append(institute.Courses, course.(models.Course))
		}

		return institute, nil

	}
}
