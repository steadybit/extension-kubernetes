// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extcommon

import "regexp"

// lineMatcher decides whether a log line is an error line. Implemented by *regexp.Regexp.
type lineMatcher interface {
	Match(line []byte) bool
}

// compileErrorMatcher returns a matcher for the error pattern. The default pattern is matched by a hand-written
// matcher with the same semantics, as the regular expression engine is by far the most expensive part of counting.
func compileErrorMatcher(pattern string) (lineMatcher, error) {
	if pattern == DefaultLogErrorPattern {
		return defaultErrorMatcher{}, nil
	}
	return regexp.Compile(pattern)
}

// defaultErrorMatcher is equivalent to DefaultLogErrorPattern: `(?i)(\b(error|fatal|panic|critical)\b|exception\b)`.
type defaultErrorMatcher struct{}

var (
	errorWordsE = []string{"error", "exception"}
	errorWordsF = []string{"fatal"}
	errorWordsP = []string{"panic"}
	errorWordsC = []string{"critical"}
)

func (defaultErrorMatcher) Match(line []byte) bool {
	for i := 0; i < len(line); i++ {
		var words []string
		// |0x20 maps ASCII upper case letters to lower case; hasFoldPrefix checks the full word anyway.
		switch line[i] | 0x20 {
		case 'e':
			words = errorWordsE
		case 'f':
			words = errorWordsF
		case 'p':
			words = errorWordsP
		case 'c':
			words = errorWordsC
		default:
			continue
		}
		for _, word := range words {
			if !hasFoldPrefix(line[i:], word) {
				continue
			}
			end := i + len(word)
			if end < len(line) && isWordByte(line[end]) {
				continue
			}
			// "exception" only needs a word boundary after it (e.g. "NullPointerException")
			if word == "exception" || i == 0 || !isWordByte(line[i-1]) {
				return true
			}
		}
	}
	return false
}

// hasFoldPrefix reports whether b starts with word (lower case ASCII letters), ignoring ASCII case. None of the
// letters of the error words have non-ASCII case folds, so this is equivalent to (?i) for them.
func hasFoldPrefix(b []byte, word string) bool {
	if len(b) < len(word) {
		return false
	}
	for j := 0; j < len(word); j++ {
		if b[j]|0x20 != word[j] {
			return false
		}
	}
	return true
}

// isWordByte matches the ASCII word characters of `\b`: [0-9A-Za-z_].
func isWordByte(c byte) bool {
	return c == '_' || ('0' <= c && c <= '9') || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
}
