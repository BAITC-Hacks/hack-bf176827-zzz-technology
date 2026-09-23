package assistant

import (
	"regexp"
)

type Answer struct {
	Text   string   `json:"answer"`
	Gids   []string `json:"gids"`
	Steps  int      `json:"steps"`
	Cached bool     `json:"cached"`
}

var gidRe = regexp.MustCompile(`\b1\d{17}\b`)
