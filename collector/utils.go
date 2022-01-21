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
