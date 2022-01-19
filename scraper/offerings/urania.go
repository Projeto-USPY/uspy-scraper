package offerings

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Projeto-USPY/uspy-backend/entity/models"
	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper"
	log "github.com/sirupsen/logrus"
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

func (sc *UraniaScraper) Process() func() (processor.Processed, error) {
	return func() (processor.Processed, error) {
		URL := fmt.Sprintf(sc.URLMask, sc.Code, sc.Since)
		resp, reader, err := scraper.Fetch(URL, http.MethodGet, nil, nil, true)

		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

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
					Professor: sc.ProfessorName,
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
				CodPes:    sc.Code,
				Professor: sc.ProfessorName,
				Years:     make([]string, 0, len(v)),
			}

			for year := range v {
				flattenedOffering.Years = append(flattenedOffering.Years, year)
			}

			offs = append(offs, flattenedOffering)
		}

		prof := models.Professor{
			CodPes:    sc.Code,
			Name:      sc.ProfessorName,
			Offerings: offs,
		}

		log.Infof("collected professor %s, num offerings: %d\n", sc.ProfessorName, len(offs))
		return prof, nil
	}
}
