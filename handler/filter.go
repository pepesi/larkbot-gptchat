package handler

import (
	"math/rand"
	"strings"

	"github.com/spf13/viper"
)

const FilterMap = "Message.FilterMap"

type KeywordsFilter struct {
	filterMap map[string][]string
}

func (f *KeywordsFilter) Filter(text string) string {
	for key, replaces := range f.filterMap {
		if strings.Contains(strings.ToLower(text), key) {
			idx := rand.Intn(len(replaces))
			return replaces[idx]
		}
	}
	return text
}

func NewFilter() *KeywordsFilter {
	f := &KeywordsFilter{
		filterMap: viper.GetStringMapStringSlice(FilterMap),
	}
	return f
}
