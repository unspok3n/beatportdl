package beatport

import (
	"encoding/json"
	"fmt"
	"strings"
)

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

func (b *Beatport) GetLabel(id int64) (*Label, error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf("/catalog/labels/%d/", id),
		nil,
		"",
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	response := &Label{}
	if err = json.NewDecoder(res.Body).Decode(response); err != nil {
		return nil, err
	}
	return response, nil
}

func (b *Beatport) GetLabelReleases(id int64, page int) (*Paginated[Release], error) {
	res, err := b.fetch(
		"GET",
		fmt.Sprintf("/catalog/labels/%d/releases/?page=%d", id, page),
		nil,
		"",
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	var response Paginated[Release]
	if err = json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &response, nil
}
