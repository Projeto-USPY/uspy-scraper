package offerings

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-backend/entity/models"
	"github.com/Projeto-USPY/uspy-scraper/scraper"
)

type UraniaOffering struct {
	Code         string `json:"coddis"`
	SubjectName  string `json:"nomdis,omitempty"`
	SubjectClass string `json:"codtur,omitempty"`
}

type UraniaHistory struct {
	NUSP    json.Number                 `json:"codpes"`
	Since   json.Number                 `json:"anoini"`
	History map[string][]UraniaOffering `json:"aulasGradPorAno"`
}

type UraniaScraper struct {
	URLMask       string
	Code          string
	Since         string
	ProfessorName string
}

var (
	DefaultOfferingURLMask = "https://uspdigital.usp.br/datausp/servicos/publico/academico/aulas_ministradas/%s/%s/0/0/br"
)

func NewUraniaScraper(code, since, name string) UraniaScraper {
	return UraniaScraper{
		URLMask:       DefaultOfferingURLMask,
		Code:          code,
		Since:         since,
		ProfessorName: name,
	}
}

func (os UraniaScraper) Start() (db.Writer, error) {
	URL := fmt.Sprintf(os.URLMask, os.Code, os.Since)
	return scraper.Start(os, URL, http.MethodGet, nil, nil)
}

func (os UraniaScraper) Scrape(reader io.Reader) (obj db.Writer, err error) {
	dec := json.NewDecoder(reader)

	var hist UraniaHistory
	if err := dec.Decode(&hist); err != nil {
		return nil, err
	}

	// we will store all offerings for each subject offered by this professor
	allOfferings := make(map[string]map[string]*models.Offering)

	for year, v := range hist.History {
		for _, data := range v {
			offering := &models.Offering{
				Professor: os.ProfessorName,
				CodPes:    hist.NUSP.String(),
				Code:      data.Code,
			}

			if len(allOfferings[offering.Code]) == 0 {
				allOfferings[offering.Code] = make(map[string]*models.Offering)
			}

			offering.Years = []string{year}
			allOfferings[offering.Code][year] = offering
		}
	}

	// flatten offerings
	offs := make([]models.Offering, 0, 5000)
	for k, v := range allOfferings {
		flattenedOffering := models.Offering{
			Code:      k,
			CodPes:    os.Code,
			Professor: os.ProfessorName,
			Years:     make([]string, 0, len(v)),
		}

		for year := range v {
			flattenedOffering.Years = append(flattenedOffering.Years, year)
		}

		offs = append(offs, flattenedOffering)
	}

	prof := models.Professor{
		CodPes:    os.Code,
		Name:      os.ProfessorName,
		Offerings: offs,
	}

	return prof, nil
}
