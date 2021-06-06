package offerings

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
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

	// since what only matters is the subject code, we will store the last time it was offered by this professor
	latestOffering := make(map[string]*models.Offering)

	for year, v := range hist.History {
		for _, data := range v {
			if _, ok := latestOffering[data.Code]; !ok {
				latestOffering[data.Code] = &models.Offering{
					Professor: os.ProfessorName,
					CodPes:    hist.NUSP.String(),
					Code:      data.Code,
					Year:      year,
				}
			} else if year > latestOffering[data.Code].Year {
				latestOffering[data.Code].Year = year
			}
		}
	}

	offs := make([]models.Offering, 0, 5000)
	for _, v := range latestOffering {
		log.Println("collected", v)
		offs = append(offs, *v)
	}

	prof := models.Professor{
		CodPes:    os.Code,
		Name:      os.ProfessorName,
		Offerings: offs,
	}

	return prof, nil
}
