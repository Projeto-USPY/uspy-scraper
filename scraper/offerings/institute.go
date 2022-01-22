package offerings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/Projeto-USPY/uspy-backend/entity/models"
	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper"
)

var (
	// {institute/department/begin/end/G|S|W}
	DefaultProfessorsURLMask = "https://uspdigital.usp.br/datausp/servicos/publico/citacoes/docentes/%s/%s/%s/%s/%s"
)

type ProfessorsList []struct {
	Code json.Number `json:"codpes"`
	Name string      `json:"nompes"`
}

type InstituteScraper struct {
	ProfessorsURLMask string

	Institute string
	Begin     string
	End       string
	Types     []string
}

func NewInstituteScraper(institute string) InstituteScraper {
	return InstituteScraper{
		ProfessorsURLMask: DefaultProfessorsURLMask,
		Institute:         institute,
		Begin:             "2010",
		End:               strconv.Itoa(time.Now().Year()), // current year
		Types:             []string{"G", "S", "W"},
	}
}

func (sc *InstituteScraper) Process(ctx context.Context) func(context.Context) (processor.Processed, error) {
	return func(context.Context) (processor.Processed, error) {
		// preprocess by getting institute departments
		depScraper := NewDepartmentsScraper(sc.Institute)
		callback := depScraper.Process()

		results, err := callback()
		if err != nil {
			return nil, err
		}

		departments := results.(DepartmentsList)
		professorTasks := make([]*processor.Task, 0)

		for _, dep := range departments {
			for _, citationsType := range sc.Types {
				professorTasks = append(professorTasks, processor.NewTask(
					log.Fields{
						"name":       "professor-task",
						"department": dep.Department,
						"category":   citationsType,
					},
					processor.QuadraticDelay,
					sc.ScrapeProfessor(dep.Department, citationsType),
					nil,
				))
			}
		}

		proc := processor.NewProcessor(
			ctx,
			log.Fields{
				"name":      "professor-processor",
				"institute": sc.Institute,
			},
			professorTasks,
			processor.Config.FixedAttempts,
			processor.Config.DelayAttempts,
		)

		depResults := proc.Run()
		offeringTasks := make([]*processor.Task, 0)

		for _, department := range depResults {
			for _, prof := range department.(ProfessorsList) {
				year := "2015"
				uraniaScraper := NewUraniaScraper(prof.Code.String(), year, prof.Name)
				offeringTasks = append(offeringTasks, processor.NewTask(
					log.Fields{
						"name":           "urania-task",
						"professor":      prof.Code,
						"professor-name": prof.Name,
					},
					processor.QuadraticDelay,
					uraniaScraper.Process(),
					nil,
				))
			}

		}

		proc = processor.NewProcessor(
			ctx,
			log.Fields{
				"name":      "offerings-processor",
				"institute": sc.Institute,
			},
			offeringTasks,
			processor.Config.FixedAttempts,
			processor.Config.DelayAttempts,
		)

		profs := proc.Run()

		// append results to institute object
		var inst models.Institute
		exists := make(map[string]struct{}) // set to check if this professor was already scraped

		for _, prof := range profs {
			professor := prof.(models.Professor)
			if _, ok := exists[professor.CodPes]; ok {
				continue
			}

			exists[professor.CodPes] = struct{}{}
			inst.Professors = append(inst.Professors, professor)

			if len(professor.Offerings) == 0 {
				log.Warnln("found no offerings for professor", professor.CodPes, professor.Name)
			}
		}

		return inst, nil
	}
}

func (sc *InstituteScraper) ScrapeProfessor(department json.Number, citationsType string) func(context.Context) (processor.Processed, error) {
	return func(context.Context) (processor.Processed, error) {
		URL := fmt.Sprintf(sc.ProfessorsURLMask, sc.Institute, department, sc.Begin, sc.End, citationsType)
		resp, reader, err := scraper.Fetch(URL, http.MethodGet, nil, nil, false)

		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		dec := json.NewDecoder(reader)

		var profs ProfessorsList

		if err := dec.Decode(&profs); err != nil {
			return nil, err
		}

		return profs, nil
	}
}
