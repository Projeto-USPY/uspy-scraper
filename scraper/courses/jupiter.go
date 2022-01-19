package courses

import (
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

func (sc *JupiterScraper) Process() func() (processor.Processed, error) {
	return func() (processor.Processed, error) {
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

		coursesHref := doc.Find("td[valign=\"top\"] a.link_gray")
		for i := 0; i < coursesHref.Length(); i++ {
			// follow every course href
			node := coursesHref.Eq(i)
			if courseCode, courseSpec, err := getCourseIdentifiers(node); err != nil {
				return nil, err
			} else {
				courseScraper := NewCourseScraper(sc.Code, courseCode, courseSpec)
				courseTasks = append(courseTasks, processor.NewTask(
					fmt.Sprintf(
						"[course-task] %s:%s",
						institute.Code,
						courseCode,
					),
					processor.QuadraticDelay,
					courseScraper.Process(),
					nil,
				))
			}
		}

		proc := processor.NewProcessor(
			fmt.Sprintf(
				"[institute-processor] %s",
				institute.Code,
			),
			courseTasks,
			processor.Config.Processor.NumWorkers,
			processor.Config.Processor.MaxAttempts,
			processor.Config.Processor.Timeout,
		)

		results := proc.Run()

		for _, course := range results {
			institute.Courses = append(institute.Courses, course.(models.Course))
		}

		log.Infof("collected %s with num courses: %d\n", institute.Name, len(institute.Courses))
		return institute, nil

	}
}
