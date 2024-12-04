package beatport

import (
	"strconv"
)

type Key struct {
	Name          string    `json:"name"`
	Letter        string    `json:"letter"`
	ChordType     ChordType `json:"chord_type"`
	CamelotNumber int       `json:"camelot_number"`
	CamelotLetter string    `json:"camelot_letter"`
	IsFlat        bool      `json:"is_flat"`
	IsSharp       bool      `json:"is_sharp"`
}

type ChordType struct {
	Name string `json:"name"`
}

func (k *Key) Display(system string) string {
	switch system {
	case "openkey":
		return k.Name
	case "openkey-short":
		var symbol string
		if k.IsSharp {
			symbol = "#"
		} else if k.IsFlat {
			symbol = "b"
		}
		var chord string
		if k.ChordType.Name == "Minor" {
			chord = "m"
		}
		return k.Letter + symbol + chord
	case "camelot":
		return strconv.Itoa(k.CamelotNumber) + k.CamelotLetter
	default:
		return ""
	}
}