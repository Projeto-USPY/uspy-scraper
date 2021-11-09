package offerings

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Projeto-USPY/uspy-backend/db"
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

func (sc DepartmentsScraper) Start() (db.Writer, error) {
	URL := fmt.Sprintf(sc.DepartmentsURLMask, sc.Institute)
	return scraper.Start(sc, URL, http.MethodGet, nil, nil, true)
}

func (sc DepartmentsScraper) Scrape(reader io.Reader) (obj db.Writer, err error) {
	dec := json.NewDecoder(reader)

	var deps DepartmentsList
	if err := dec.Decode(&deps); err != nil {
		return nil, err
	}

	return deps, nil
}
