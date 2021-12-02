package courses

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity/models"
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
func (sc CourseScraper) Start() (db.Writer, error) {
	URL := fmt.Sprintf(sc.URLMask, sc.InstituteCode, sc.Code, sc.Specialization)
	return scraper.Start(sc, URL, http.MethodGet, nil, nil, true)
}

func (sc CourseScraper) Scrape(reader io.Reader) (db.Writer, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	course := models.Course{
		Name:           doc.Find("td > font:nth-child(2) > span").Last().Text(),
		Code:           sc.Code,
		Specialization: sc.Specialization,
		Subjects:       make([]models.Subject, 0, 1000),
		SubjectCodes:   make(map[string]string),
	}

	// Get Subjects
	sections := doc.Find("tr[bgcolor='#658CCF']") // Finds section "Disciplinas Obrigatórias"

	if sections.Length() == 0 {
		return nil, ErrorCourseNoSubjects
	}

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
				obj, err := subjectScraper.Start()

				if err != nil {
					return nil, err
				}

				subject := obj.(models.Subject)

				requirementLists := make(map[string][]models.Requirement)
				requirements := []models.Requirement{}
				groupIndex := 0

				// Get requirements of subject
				for l := 0; l < rows.Length(); l++ {
					row := rows.Eq(l)

					if row.Has("b").Length() > 0 { // "row" is an "or"
						groupIndex++
						requirementLists[strconv.Itoa(groupIndex)] = requirements
						requirements = []models.Requirement{}
					} else if row.Has(".txt_arial_8pt_red").Length() > 0 { // "row" is an actual requirement
						reqText := row.Children().Eq(0).Text()
						strongText := row.Children().Eq(1).Text()

						reqSplitText := strings.SplitN(reqText, "-", 2)
						if len(reqSplitText) < 2 {
							return nil, errors.New("couldn't parse requirement")
						}

						reqCode, reqName := strings.TrimSpace(reqSplitText[0]), strings.TrimSpace(reqSplitText[1])

						if strings.Contains(strongText, "Requisito") {
							requirements = append(requirements, models.Requirement{
								Subject: reqCode,
								Name:    reqName,
								Strong:  !strings.Contains(strongText, "fraco"),
							})
						}

					} else { // "row" is an empty <tr>
						break
					}
				}

				if len(requirements) > 0 {
					groupIndex++
					requirementLists[strconv.Itoa(groupIndex)] = requirements
				}

				subject.Requirements = requirementLists
				subject.Optional = optional
				subject.Semester, _ = strconv.Atoi(strings.Split(period.Find(".txt_arial_8pt_black").Text(), "º")[0])
				subject.TrueRequirements = make([]models.Requirement, 0)

				count := make(map[string]int)
				for _, group := range subject.Requirements {
					for _, s := range group {
						count[s.Subject]++
						if count[s.Subject] == len(subject.Requirements) {
							subject.TrueRequirements = append(subject.TrueRequirements, s)
						}
					}
				}

				course.Subjects = append(course.Subjects, subject)
			}
		}

		optional = true // after the first section, all subjects are optional
	}

	log.Println("collected", course.Name, "with num subjects:", len(course.Subjects))
	for _, s := range course.Subjects {
		course.SubjectCodes[s.Code] = s.Name
	}

	return course, nil
}
