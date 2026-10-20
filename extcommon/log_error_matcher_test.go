// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extcommon

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var matcherCorpus = []string{
	"", "error", "ERROR", "Error:", "errors", "error_count=0", "_error", "1error", "error1", "my-error", "err",
	"stderr redirected", "level=error msg=x", `{"level":"error"}`, "[ERROR] x", "FATAL", "fatalistic", "nonfatal",
	"panic: boom", "panicked", "Critical", "CRITICAL!", "criticality", "exception", "NullPointerException: x",
	"Exception in thread", "exceptional", "exceptions", "EXCEPTION_X", "e", "erro", "eRrOr", "x error y",
	"\xffERROR\xff", "Ünïcödé error ✓", "ſtrange", "Kelvin K error", "panic\x00", "ex ception", "eexception",
	"INFO request served", "errorerror error", "warn: fatal_error", "fatal-error",
}

func TestDefaultErrorMatcherIsEquivalentToThePattern(t *testing.T) {
	pattern := regexp.MustCompile(DefaultLogErrorPattern)
	for _, line := range matcherCorpus {
		assert.Equal(t, pattern.MatchString(line), defaultErrorMatcher{}.Match([]byte(line)), "line %q", line)
	}
}

func FuzzDefaultErrorMatcher(f *testing.F) {
	for _, line := range matcherCorpus {
		f.Add(line)
	}
	pattern := regexp.MustCompile(DefaultLogErrorPattern)
	f.Fuzz(func(t *testing.T, line string) {
		if pattern.MatchString(line) != (defaultErrorMatcher{}).Match([]byte(line)) {
			t.Fatalf("matcher differs from %s for %q", DefaultLogErrorPattern, line)
		}
	})
}

func TestCompileErrorMatcher(t *testing.T) {
	m, err := compileErrorMatcher(DefaultLogErrorPattern)
	require.NoError(t, err)
	assert.IsType(t, defaultErrorMatcher{}, m)

	m, err = compileErrorMatcher("boom")
	require.NoError(t, err)
	assert.IsType(t, &regexp.Regexp{}, m)
	assert.True(t, m.Match([]byte("a boom")))

	_, err = compileErrorMatcher("(")
	require.Error(t, err)
}
