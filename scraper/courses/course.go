package courses

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/Projeto-USPY/uspy-backend/entity/models"
	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper"
	"github.com/PuerkitoBio/goquery"
)

var (
	ErrorCourseNotExist   = errors.New("could not fetch course in institute page")
	ErrorCourseNoSubjects = errors.New("could not fetch subjects in course page")

	DefaultCourseURLMask = "https://uspdigital.usp.br/jupiterweb/listarGradeCurricular?codcg=%s&codcur=%s&codhab=%s&tipo=N"
)

type CourseScraper struct {
	URLMask        string
	InstituteCode  string
	Code           string
	Specialization string
}

func NewCourseScraper(institute, course, spec string) CourseScraper {
	return CourseScraper{
		URLMask:        DefaultCourseURLMask,
		Code:           course,
		Specialization: spec,
		InstituteCode:  institute,
	}
}

func (cs *CourseScraper) Process() func() (processor.Processed, error) {
	return func() (processor.Processed, error) {
		URL := fmt.Sprintf(cs.URLMask, cs.InstituteCode, cs.Code, cs.Specialization)

		resp, reader, err := scraper.Fetch(URL, http.MethodGet, nil, nil, true)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromReader(reader)

		if err != nil {
			return nil, err
		}

		course := models.Course{
			Name:           doc.Find("td > font:nth-child(2) > span").Last().Text(),
			Code:           cs.Code,
			Specialization: cs.Specialization,
			Subjects:       make([]models.Subject, 0, 1000),
			SubjectCodes:   make(map[string]string),
		}

		// Get Subjects
		sections := doc.Find("tr[bgcolor='#658CCF']") // Finds section "Disciplinas Obrigatórias"

		if sections.Length() == 0 {
			return nil, ErrorCourseNoSubjects
		}

		subjectTasks := make([]*processor.Task, 0)

		optional := false
		// For each section (obrigatorias, eletivas)
		for i := 0; i < sections.Length(); i++ {
			s := sections.Eq(i)
			periods := s.NextUntil("tr[bgcolor='#658CCF']").Filter("tr[bgcolor='#CCCCCC']") // Periods section, for each subject

			// Get each semester/period
			for j := 0; j < periods.Length(); j++ {
				period := periods.Eq(j)

				subjects := period.NextUntilSelection(periods.Union(sections)).Find("a")

				// Get subjects in current section and semester
				for k := 0; k < subjects.Length(); k++ { // for each <tr>
					subjectNode := subjects.Eq(k).Closest("tr")
					rows := subjectNode.NextUntilSelection(subjects.Union(periods).Union(sections))

					subjectObj := subjectNode.Find("a")
					subjectScraper := NewSubjectScraper(strings.TrimSpace(subjectObj.Text()), course.Code, course.Specialization)

					// create subject task
					subjectTasks = append(subjectTasks, processor.NewTask(
						fmt.Sprintf( // task id = subject:course:specialization
							"[subject-task] %s:%s:%s",
							strings.TrimSpace(subjectObj.Text()),
							course.Code,
							course.Specialization,
						),
						processor.QuadraticDelay,
						subjectScraper.Process(period, rows, optional),
						nil,
					))
				}

			}

			optional = true // after the first section, all subjects are optional
		}

		proc := processor.NewProcessor(
			fmt.Sprintf(
				"[subject-processor] %s",
				strings.ToLower(course.Name),
			),
			subjectTasks,
			processor.Config.Processor.NumWorkers,
			processor.Config.Processor.MaxAttempts,
			processor.Config.Processor.Timeout,
		)

		results := proc.Run()

		for _, subject := range results {
			course.Subjects = append(course.Subjects, subject.(models.Subject))
		}

		for _, s := range course.Subjects {
			course.SubjectCodes[s.Code] = s.Name
		}

		log.Infof("collected %s with num subjects: %d\n", course.Name, len(course.Subjects))
		return course, nil
	}
}
