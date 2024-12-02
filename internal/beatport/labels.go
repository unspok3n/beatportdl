package beatport

import "strings"

type Label struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func (l Label) NameSanitized() string {
	charsToRemove := []string{"/", "\\", "?", "\"", "|", "*", ":", "<", ">", "."}
	for _, char := range charsToRemove {
		l.Name = strings.Replace(l.Name, char, "", -1)
	}
	return l.Name
}
