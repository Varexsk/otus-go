package hw02unpackstring

import (
	"errors"
	"strconv"
	"strings"
)

var ErrInvalidString = errors.New("invalid string")

func Unpack(s string) (string, error) {
	r := []rune(s)

	var b strings.Builder
	b.Grow(32)

	var escaped bool
	for i := 0; i < len(r); i++ {
		_, isNumber := IsNumber(r[i])
		if isNumber && !escaped {
			return "", ErrInvalidString
		}

		var nextRune rune
		if i < len(r)-1 {
			nextRune = r[i+1]
		}

		if r[i] == '\\' && !escaped {
			escaped = true
			continue
		}

		n, nextNumber := IsNumber(nextRune)

		if nextNumber {
			b.WriteString(strings.Repeat(string(r[i]), n))
			i++
		} else {
			b.WriteRune(r[i])
		}
		
		escaped = false
	}

	return b.String(), nil
}

func IsNumber(r rune) (int, bool) {
	n, err := strconv.Atoi(string(r))
	if err != nil {
		return 0, false
	}
	return n, true
}
