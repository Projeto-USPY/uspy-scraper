package courses

import (
	"encoding/json"
	"testing"
)

func TestNewJupiterScraper(t *testing.T) {
	instituteSc := NewJupiterScraper("55")
	if institute, err := instituteSc.Start(); err != nil {
		t.Fatal(err)
	} else {
		if bytes, err := json.MarshalIndent(&institute, "", "    "); err != nil {
			t.Fatal(err)
		} else {
			t.Log(string(bytes))
		}
	}
}
