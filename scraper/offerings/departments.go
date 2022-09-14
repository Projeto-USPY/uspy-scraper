package offerings

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Projeto-USPY/uspy-backend/db"
	"github.com/Projeto-USPY/uspy-scraper/processor"
	"github.com/Projeto-USPY/uspy-scraper/scraper"
)

var (
	// {institute}
	DefaultDepartmentsURLMask = "https://uspdigital.usp.br/datausp/servicos/publico/listagem/departamentos/%s"

	// {year, institute}
	fallbackDepartmentsURLMask = "https://uspdigital.usp.br/datausp/servicos/publico/indicadores/unidade/graficos/docenteDepartamento/%d/%s"
)

type DepartmentsList []struct {
	Department json.Number `json:"codset"`
}

type fallbackDepartmentsList []struct {
	Department json.Number `json:"codigoSetor"`
}

func (d DepartmentsList) Insert(_ db.Database, _ string) error { return nil }
func (d DepartmentsList) Update(_ db.Database, _ string) error { return nil }

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

func getDepartments(mask string, resultsPointer interface{}, params ...interface{}) error {
	URL := fmt.Sprintf(mask, params...)

	resp, reader, err := scraper.Fetch(URL, http.MethodGet, nil, nil, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(reader)
	return dec.Decode(&resultsPointer)
}

func (sc *DepartmentsScraper) Process() func() (processor.Processed, error) {
	return func() (processor.Processed, error) {
		var results DepartmentsList
		var fallback fallbackDepartmentsList

		if err := getDepartments(sc.DepartmentsURLMask, &results, sc.Institute); err != nil {
			return nil, err
		}

		if len(results) == 0 { // use fallback
			year := time.Now().Year()
			if err := getDepartments(fallbackDepartmentsURLMask, &fallback, year, sc.Institute); err != nil {
				return nil, err
			}

			for _, f := range fallback {
				results = append(results, struct {
					Department json.Number "json:\"codset\""
				}{f.Department})
			}
		}

		return results, nil
	}
}
