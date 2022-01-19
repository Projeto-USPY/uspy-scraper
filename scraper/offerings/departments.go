package offerings

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper"
)

var (
	// {institute}
	DefaultDepartmentsURLMask = "https://uspdigital.usp.br/datausp/servicos/publico/listagem/departamentos/%s"
)

type DepartmentsList []struct {
	Department json.Number `json:"codset"`
}

func (d DepartmentsList) Insert(_ db.Env, _ string) error { return nil }
func (d DepartmentsList) Update(_ db.Env, _ string) error { return nil }

type DepartmentsScraper struct {
	DepartmentsURLMask string
	Institute          string
}

func NewDepartmentsScraper(institute string) DepartmentsScraper {
	return DepartmentsScraper{
		DepartmentsURLMask: DefaultDepartmentsURLMask,
		Institute:          institute,
	}
}

func (sc *DepartmentsScraper) Process() func() (processor.Processed, error) {
	return func() (processor.Processed, error) {
		URL := fmt.Sprintf(sc.DepartmentsURLMask, sc.Institute)

		resp, reader, err := scraper.Fetch(URL, http.MethodGet, nil, nil, true)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		dec := json.NewDecoder(reader)

		var deps DepartmentsList
		if err := dec.Decode(&deps); err != nil {
			return nil, err
		}

		return deps, nil
	}
}
