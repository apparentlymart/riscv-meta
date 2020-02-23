package main

import (
	"strings"
	"unicode"
)

func makeIdentUnderscores(inp string) string {
	var b strings.Builder
	for i, r := range inp {
		switch {
		case unicode.IsDigit(r):
			if i == 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r)
		case unicode.IsLetter(r):
			b.WriteString(strings.ToLower(string(r)))
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func makeIdentTitle(inp string) string {
	var b strings.Builder
	nextUpper := true
	for i, r := range inp {
		switch {
		case unicode.IsDigit(r):
			if i == 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r)
			nextUpper = true
		case unicode.IsLetter(r):
			if nextUpper {
				b.WriteString(strings.ToUpper(string(r)))
			} else {
				b.WriteString(strings.ToLower(string(r)))
			}
			nextUpper = false
		default:
			nextUpper = true
		}
	}
	return b.String()
}
