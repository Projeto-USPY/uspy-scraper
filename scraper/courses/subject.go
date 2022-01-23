package courses

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/Projeto-USPY/uspy-backend/entity/models"
	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper"
	"github.com/PuerkitoBio/goquery"
)

var (
	DefaultSubjectRoot = "https://uspdigital.usp.br/jupiterweb/"
)

type SubjectScraper struct {
	URL  string
	Code string
}

func NewSubjectScraper(subjectCode, subjectURL string) SubjectScraper {
	return SubjectScraper{
		Code: subjectCode,
		URL:  subjectURL,
	}
}

func getRequirements(period, rows *goquery.Selection, optional bool, subject *models.Subject) error {
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
				return errors.New("couldn't parse requirement")
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

	return nil
}

func getDescription(doc *goquery.Document) (string, error) {
	var objetivosNode *goquery.Selection = nil
	bold := doc.Find("b")

	for i := 0; i < bold.Length(); i++ {
		s := bold.Eq(i)
		text := s.Text() // get inner html

		if strings.TrimSpace(text) == "Objetivos" { // found
			objetivosNode = s
		}
	}

	if objetivosNode == nil {
		return "", nil
	}

	objetivosTr := objetivosNode.Closest("tr") // get tr parent
	descriptionTr := objetivosTr.Next()        // tr with description is next <tr>

	desc := strings.TrimSpace(descriptionTr.Text())
	return desc, nil
}

func getClassCredits(search *goquery.Selection) (int, error) {
	classCredits := strings.TrimSpace(search.Eq(0).Text())
	class, err := strconv.Atoi(classCredits)

	if err != nil {
		return -1, err
	}

	return class, nil
}

func getAssignCredits(search *goquery.Selection) (int, error) {
	assignCredits := strings.TrimSpace(search.Eq(1).Text())
	assign, err := strconv.Atoi(assignCredits)

	if err != nil {
		return -1, err
	}

	return assign, nil
}

func getTotalHours(search *goquery.Selection) (string, error) {
	totalHours := strings.Trim(search.Eq(2).Text(), " \n\t")
	space, err := regexp.Compile(`\s+`)
	if err != nil {
		return "", err
	}

	total := space.ReplaceAllString(totalHours, " ")
	return total, nil
}

func (sc *SubjectScraper) Process(period, rows *goquery.Selection, optional bool) func(context.Context) (processor.Processed, error) {
	return func(context.Context) (processor.Processed, error) {
		resp, reader, err := scraper.Fetch(sc.URL, http.MethodGet, nil, nil, true)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		rg, err := regexp.Compile(`codcur=([0-9]+)&codhab=([0-9]+)`)
		if err != nil {
			return nil, fmt.Errorf("failed to get code, course and specialization from fallback URL: %s", err.Error())
		}

		matches := rg.FindAllStringSubmatch(sc.URL, -1)
		if len(matches) != 1 || len(matches[0]) != 3 {
			return nil, errors.New("failed to get code, course and specialization from fallback URL")
		}

		course, specialization := matches[0][1], matches[0][2]

		doc, err := goquery.NewDocumentFromReader(reader)
		if err != nil {
			return nil, err
		}

		fullName := doc.Find("span.txt_arial_10pt_black > b").Text()
		fields := strings.SplitN(fullName, "-", 2)

		if len(fields) < 2 {
			return nil, fmt.Errorf("could not get subject name from %s, this is unexpected", sc.URL)
		}

		name := strings.TrimSpace(fields[1])

		subject := models.Subject{
			Code:           sc.Code,
			CourseCode:     course,
			Specialization: specialization,
			Name:           name,
			Stats: map[string]int{
				"total":    0,
				"worth_it": 0,
			},
		}

		if description, err := getDescription(doc); err == nil {
			subject.Description = description
		} else {
			return nil, err
		}

		search := doc.Find("tr[valign=\"TOP\"][align=\"LEFT\"] > td > font > span[class=\"txt_arial_8pt_gray\"]")
		if class, err := getClassCredits(search); err == nil {
			subject.ClassCredits = class
		} else {
			return nil, err
		}

		if assign, err := getAssignCredits(search); err == nil {
			subject.AssignCredits = assign
		} else {
			return nil, err
		}

		if total, err := getTotalHours(search); err == nil {
			subject.TotalHours = total
		} else {
			return nil, err
		}

		getRequirements(period, rows, optional, &subject)
		return subject, nil
	}
}
