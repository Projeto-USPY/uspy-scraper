package professors

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
	"github.com/Projeto-USPY/uspy-scraper/scraper/offerings"
)

var (
	// {institute/department/begin/end/G|S|W}
	DefaultProfessorsURLMask = "https://uspdigital.usp.br/datausp/servicos/publico/citacoes/docentes/%s/%s/%s/%s/%s"
)

type ProfessorsList []struct {
	Code json.Number `json:"codpes"`
	Name string      `json:"nompes"`
}

type InstituteProfessorsScraper struct {
	ProfessorsURLMask string

	Institute string
	Begin     string
	End       string
	Types     []string
}

func NewInstituteProfessorsScraper(institute string) InstituteProfessorsScraper {
	return InstituteProfessorsScraper{
		ProfessorsURLMask: DefaultProfessorsURLMask,
		Institute:         institute,
		Begin:             "2010",
		End:               strconv.Itoa(time.Now().Year()), // current year
		Types:             []string{"G", "S", "W"},
	}
}

func (sc *InstituteProfessorsScraper) Process(ctx context.Context) func(context.Context) (processor.Processed, error) {
	return func(context.Context) (processor.Processed, error) {
		// preprocess by getting institute departments
		depScraper := offerings.NewDepartmentsScraper(sc.Institute)
		callback := depScraper.Process()

		results, err := callback()
		if err != nil {
			return nil, err
		}

		departments := results.(offerings.DepartmentsList)
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

		// append results to institute object
		var inst models.Institute = models.Institute{
			Code: sc.Institute,
		}

		for _, department := range depResults {
			for _, prof := range department.(ProfessorsList) {
				inst.Professors = append(inst.Professors,
					models.Professor{
						CodPes: prof.Code.String(),
						Name:   prof.Name,
					},
				)
			}

		}

		return inst, err
	}
}

func (sc *InstituteProfessorsScraper) ScrapeProfessor(department json.Number, citationsType string) func(context.Context) (processor.Processed, error) {
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
