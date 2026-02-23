package scraper

import "strings"

var registry = map[string]Scraper{}

func Register(s Scraper) {
	registry[strings.ToLower(s.Name())] = s
}

func Get(name string) (Scraper, bool) {
	s, ok := registry[strings.ToLower(name)]
	return s, ok
}
