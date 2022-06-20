// Package wildcards implements helper to compare text with wildcards in tests.
//
// Inspired by PhpUnit "assertStringMatchesFormat".
//
//   Supported wildcards:
//     %e: Represents a directory separator, for example / on Linux.
//     %s: One or more of anything (character or white space) except the end of line character.
//     %S: Zero or more of anything (character or white space) except the end of line character.
//     %a: One or more of anything (character or white space) including the end of line character.
//     %A: Zero or more of anything (character or white space) including the end of line character.
//     %w: Zero or more white space characters.
//     %i: A signed integer value, for example +3142, -3142.
//     %d: An unsigned integer value, for example 123456.
//     %x: One or more hexadecimal character. That is, characters in the range 0-9, a-f, A-F.
//     %f: A floating point number, for example: 3.142, -3.142, 3.142E-10, 3.142e+10.
//     %c: A single character of any sort.
//     %%: A literal percent character: %.
package wildcards

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"
)

// Assert compares two texts and allows using wildcards in expected value, see ToRegexp function.
func Assert(t assert.TestingT, expected string, actual string, msgAndArgs ...interface{}) {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)

	// Replace NBSP with space
	actual = strings.ReplaceAll(actual, " ", " ")

	// Remove \r chars
	actual = strings.ReplaceAll(actual, "\r", "")

	// Assert
	if len(expected) == 0 {
		assert.Equal(t, expected, actual, msgAndArgs...)
	} else {
		expectedRegexp := ToRegexp(strings.TrimSpace(expected))
		diff := difflib.UnifiedDiff{
			A: difflib.SplitLines(EscapeWhitespaces(expected)),
			B: difflib.SplitLines(EscapeWhitespaces(actual)),
		}
		diffStr, _ := difflib.GetUnifiedDiffString(diff)
		diffStr = cleanDiffOutput(diffStr)
		r := regexp.MustCompile("^" + expectedRegexp + "$")
		if !r.MatchString(actual) {
			assert.Fail(t, fmt.Sprintf("Diff:\n-----\n%s-----\nActual:\n-----\n%s\n-----\nExpected:\n-----\n%v\n-----\n", diffStr, actual, expected), msgAndArgs...)
		}
	}
}

// ToRegexp converts string with wildcards to regexp, so it can be used in assert.Regexp.
func ToRegexp(input string) string {
	input = regexp.QuoteMeta(input)
	re := regexp.MustCompile(`%.`)
	return re.ReplaceAllStringFunc(input, func(s string) string {
		// Inspired by PhpUnit "assertStringMatchesFormat"
		// https://phpunit.readthedocs.io/en/9.5/assertions.html#assertstringmatchesformat
		switch s {
		// %e: Represents a directory separator, for example / on Linux.
		case `%e`:
			return regexp.QuoteMeta(string(os.PathSeparator)) // nolint forbidigo
		// %s: One or more of anything (character or white space) except the end of line character.
		case `%s`:
			return `.+`
		// %S: Zero or more of anything (character or white space) except the end of line character.
		case `%S`:
			return `.*`
		// %a: One or more of anything (character or white space) including the end of line character.
		case `%a`:
			return `(.|\n)+`
		// %A: Zero or more of anything (character or white space) including the end of line character.
		case `%A`:
			return `(.|\n)*`
		// %w: Zero or more white space characters.
		case `%w`:
			return `\s*`
		// %i: A signed integer value, for example +3142, -3142.
		case `%i`:
			return `(\+|\-)\d+`
		// %d: An unsigned integer value, for example 123456.
		case `%d`:
			return `\d+`
		// %x: One or more hexadecimal character. That is, characters in the range 0-9, a-f, A-F.
		case `%x`:
			return `[0-9a-zA-Z]+`
		// %f: A floating point number, for example: 3.142, -3.142, 3.142E-10, 3.142e+10.
		case `%f`:
			return `[-+]?[0-9]*\.?[0-9]+([eE][-+]?[0-9]+)?`
		// %c: A single character of any sort.
		case `%c`:
			return `.`
		// %%: A literal percent character: %.
		case `%%`:
			return `%`
		}

		return s
	})
}

// EscapeWhitespaces escapes all whitespaces except new line -> for clearer difference in diff output.
func EscapeWhitespaces(input string) string {
	re := regexp.MustCompile(`\s`)
	return re.ReplaceAllStringFunc(input, func(s string) string {
		switch s {
		case "\n":
			return s
		case "\t":
			return `→→→→`
		case " ":
			return `␣`
		default:
			return strings.Trim(strconv.Quote(s), `"`)
		}
	})
}

// cleanDiffOutput - if text doesn't match wildcards, then diff between <wildcards/text> is printed.
// So we have to remove diff blocks that are false positive.
//
// Example of diff block that should be omitted:
// 	@@ -4 +4 @@
//	-Foo:␣%s
//	+Foo:␣bar4
func cleanDiffOutput(in string) string {
	var out strings.Builder
	for _, block := range regexp.MustCompile(`(?m)^@@`).Split(in, -1) {
		// Skip first line, eg. "@@ -4 +4 @@"
		_, content, _ := strings.Cut(block, "\n")
		if content == "" {
			continue
		}

		// Separate expected and actual block, find first "+" at line beginning.
		var actual, expected string
		parts := regexp.MustCompile(`(?m)^+`).Split(content, 2)

		// Remove "-" from each line in expected block, for example "-Foo:␣%s" -> "Foo:␣%s"
		expected = regexp.MustCompile(`(?m)^-`).ReplaceAllString(parts[0], "")

		// Remove "+" from each line in actual block, for example "+Foo:␣bar4" -> "Foo:␣bar4"
		if len(parts) > 1 {
			actual = regexp.MustCompile(`(?m)^\+`).ReplaceAllString(parts[1], "")
		}

		// Compare expected and actual, for example "Foo:␣%s" and "Foo:␣bar4"
		if !regexp.MustCompile("^" + ToRegexp(expected) + "$").MatchString(actual) {
			// Keep block with difference
			out.WriteString("@@" + block)
		}
	}
	return out.String()
}
