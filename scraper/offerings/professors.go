package offerings

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/Projeto-USPY/uspy-backend/db"
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

func (d ProfessorsList) Insert(_ db.Env, _ string) error { return nil }
func (d ProfessorsList) Update(_ db.Env, _ string) error { return nil }

type ProfessorScraper struct {
	ProfessorsURLMask string

	Institute string
	Begin     string
	End       string
	Types     []string
}

func NewProfessorScraper(institute string) ProfessorScraper {
	return ProfessorScraper{
		ProfessorsURLMask: DefaultProfessorsURLMask,
		Institute:         institute,
		Begin:             "2010",
		End:               strconv.Itoa(time.Now().Year()), // current year
		Types:             []string{"G", "S", "W"},
	}
}

func (sc *ProfessorScraper) Process() func() (processor.Processed, error) {
	return func() (processor.Processed, error) {
		// preprocess by getting institute departments
		depScraper := NewDepartmentsScraper(sc.Institute)
		callback := depScraper.Process()

		departments, err := callback()
		if err != nil {
			return ProfessorScraper{}, err
		}

		professorTasks := make([]*processor.Task, 0)

		for _, dep := range departments.(DepartmentsList) {
			log.Debugln("scrapping department", dep.Department)
			for _, citationsType := range sc.Types {
				professorTasks = append(professorTasks, processor.NewTask(
					fmt.Sprintf(
						"[professor-task] %s:%s",
						dep.Department,
						citationsType,
					),
					processor.QuadraticDelay,
					sc.ScrapeProfessor(dep.Department, citationsType),
					nil,
				))
			}
		}

		proc := processor.NewProcessor(
			fmt.Sprintf("[professor-processor] %s", sc.Institute),
			professorTasks,
			processor.Config.Processor.NumWorkers,
			processor.Config.Processor.MaxAttempts,
			processor.Config.Processor.Timeout,
		)

		depResults := proc.Run()
		offeringTasks := make([]*processor.Task, 0)

		for _, department := range depResults {
			for _, prof := range department.(ProfessorsList) {
				uraniaScraper := NewUraniaScraper(prof.Code.String(), "2015", prof.Name)
				offeringTasks = append(offeringTasks, processor.NewTask(
					fmt.Sprintf(
						"[offering-task] %s:%s",
						prof.Code,
						strings.ReplaceAll(strings.ToLower(prof.Name), " ", "_"),
					),
					processor.QuadraticDelay,
					uraniaScraper.Process(),
					nil,
				))
			}

		}

		proc = processor.NewProcessor(
			fmt.Sprintf("[offerings-processor] %s", sc.Institute),
			offeringTasks,
			processor.Config.Processor.NumWorkers,
			processor.Config.Processor.MaxAttempts,
			processor.Config.Processor.Timeout,
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

		log.Infof("collected institute %s, num professors: %d\n", inst.Code, len(profs))
		return inst, nil
	}
}

func (sc *ProfessorScraper) ScrapeProfessor(department json.Number, citationsType string) func() (processor.Processed, error) {
	return func() (processor.Processed, error) {
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
