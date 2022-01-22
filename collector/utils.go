package collector

import "strings"

func parseInstitutesFromQuery(query map[string][]string) []string {
	var instituteCodes []string
	if len(query["institute"]) > 0 {
		codes := strings.Split(query["institute"][0], ",")
		institutes := make([]string, 0, len(codes))
		institutes = append(institutes, codes...)
		instituteCodes = institutes
	}

	return instituteCodes
}

func parseSkipInstitutesFromQuery(query map[string][]string) map[string]bool {
	skip := make(map[string]bool)
	if len(query["skip"]) > 0 {
		codes := strings.Split(query["skip"][0], ",")
		for _, code := range codes {
			skip[code] = true
		}
	}

	return skip
}
