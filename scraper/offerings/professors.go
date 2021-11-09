package offerings

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity/models"
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

func (sc ProfessorScraper) Start() (db.Writer, error) {
	// get departments
	log.Println("getting departments for institute", sc.Institute)

	depScraper := NewDepartmentsScraper(sc.Institute)
	depResults, err := depScraper.Start()

	if err != nil {
		return nil, err
	}

	var inst models.Institute
	exists := make(map[string]struct{}) // set to check if this professor was already scraped

	for _, dep := range depResults.(DepartmentsList) {
		log.Println("scrapping department", dep.Department)
		for _, citationsType := range sc.Types {
			URL := fmt.Sprintf(sc.ProfessorsURLMask, sc.Institute, dep.Department, sc.Begin, sc.End, citationsType)
			profResults, err := scraper.Start(sc, URL, http.MethodGet, nil, nil, false)
			if err != nil {
				return nil, err
			}

			// append results to institute object
			for _, prof := range profResults.(models.Institute).Professors {
				if _, ok := exists[prof.CodPes]; ok {
					continue
				}

				exists[prof.CodPes] = struct{}{}
				inst.Professors = append(inst.Professors, prof)

				if len(prof.Offerings) == 0 {
					log.Println("found no offerings for professor", prof.CodPes, prof.Name)
				}
			}
		}
	}

	return inst, nil
}

func (sc ProfessorScraper) Scrape(reader io.Reader) (obj db.Writer, err error) {
	dec := json.NewDecoder(reader)

	var profs ProfessorsList

	if err := dec.Decode(&profs); err != nil {
		return nil, err
	}

	var inst models.Institute
	exists := make(map[string]struct{}) // set to check if this professor was already scraped

	for _, prof := range profs {
		if _, ok := exists[prof.Code.String()]; ok {
			continue
		}

		exists[prof.Code.String()] = struct{}{}

		uraniaSc := NewUraniaScraper(prof.Code.String(), "2015", prof.Name)
		result, err := uraniaSc.Start()

		if err != nil {
			return nil, err
		}

		inst.Professors = append(inst.Professors, result.(models.Professor))
	}

	return inst, nil
}
