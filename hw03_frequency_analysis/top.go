package hw03frequencyanalysis

import (
	"sort"
	"strings"
)

type kv struct {
	key   string
	value int
}

func Top10(s string) []string {
	wmap := make(map[string]int)

	words := strings.Fields(s)

	for i := range words {
		wmap[words[i]]++
	}

	sl := make([]kv, 0, len(wmap))

	for k, v := range wmap {
		sl = append(sl, kv{key: k, value: v})
	}

	sort.Slice(sl, func(i, j int) bool {
		if sl[i].value == sl[j].value {
			return sl[i].key < sl[j].key
		}
		return sl[i].value > sl[j].value
	})

	result := make([]string, 0, 10)

	for i, v := range sl {
		if i >= 10 {
			break
		}
		result = append(result, v.key)
	}

	return result
}
