package diff

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"
)

// wildcardMatcher defines a function to match a wildcard against a value.
type wildcardMatcher func(expected string, actual any) bool

// wildcardMatchers is a registry of simple wildcards that require special validation.
type differ struct {
	wildcardMatchers map[string]wildcardMatcher
}

func newDiffer() *differ {
	matchers := make(map[string]wildcardMatcher)
	// %i: A signed integer value, for example +3142, -3142.
	// %d: An unsigned integer value, for example 123456.
	matchers["%i"] = isNumber
	matchers["%d"] = isNumber
	matchers["%f"] = isNumber
	// %s: A string value.
	// %S: A string value or null.
	matchers["%s"] = isString
	matchers["%S"] = isString
	matchers["%a"] = isString
	matchers["%A"] = isString
	matchers["%w"] = isString
	matchers["%x"] = isString
	matchers["%c"] = isString
	// %datetime: A date-time string in RFC3339 format.
	// %date: A date string in RFC3339 format.
	matchers["%datetime"] = isDateTime
	matchers["%date"] = isDate
	return &differ{wildcardMatchers: matchers}
}

// CompareJSON compares two JSON strings and allows using wildcards in expected value.
func CompareJSON(expected, actual string) error {
	var expectedData, actualData any
	if err := json.Unmarshal([]byte(expected), &expectedData); err != nil {
		return fmt.Errorf("cannot unmarshal expected JSON: %w", err)
	}
	if err := json.Unmarshal([]byte(actual), &actualData); err != nil {
		return fmt.Errorf("cannot unmarshal actual JSON: %w", err)
	}

	d := newDiffer()
	return d.compare(expectedData, actualData, "")
}

func (d *differ) compare(expected, actual any, path string) error {
	// Handle wildcards: if expected is a string, it may be a wildcard.
	if expectedStr, ok := expected.(string); ok {
		// Is it a special wildcard with type validation?
		if matcher, found := d.wildcardMatchers[expectedStr]; found {
			if matcher(expectedStr, actual) {
				return nil // Matched
			}
			// Matcher failed
			return diffError(expected, actual, path)
		}

		// Is it a general regexp wildcard?
		if strings.Contains(expectedStr, "%") {
			// Compare actual value as a string
			return compareStrings(expectedStr, stringify(actual), path)
		}
	}

	// Continue with original comparison logic if no wildcard was matched.
	expectedValue := reflect.ValueOf(expected)
	actualValue := reflect.ValueOf(actual)

	if expectedValue.Kind() != actualValue.Kind() {
		return diffError(expected, actual, path)
	}

	switch expectedValue.Kind() {
	case reflect.Map:
		return d.compareMaps(expectedValue, actualValue, path)
	case reflect.Slice:
		return d.compareSlices(expectedValue, actualValue, path)
	case reflect.String:
		return compareStrings(expectedValue.String(), actualValue.String(), path)
	default:
		if !reflect.DeepEqual(expected, actual) {
			return diffError(expected, actual, path)
		}
	}

	return nil
}

func (d *differ) compareMaps(expected, actual reflect.Value, path string) error {
	expectedMap := expected.Interface().(map[string]any)
	actualMap := actual.Interface().(map[string]any)

	for k, v := range expectedMap {
		newPath := k
		if path != "" {
			newPath = path + "." + k
		}
		if actualV, ok := actualMap[k]; ok {
			if err := d.compare(v, actualV, newPath); err != nil {
				return err
			}
		} else {
			return diffError(v, nil, newPath)
		}
	}

	// Check for extra keys in actual map
	for k := range actualMap {
		if _, ok := expectedMap[k]; !ok {
			newPath := k
			if path != "" {
				newPath = path + "." + k
			}
			return diffError(nil, actualMap[k], newPath)
		}
	}

	return nil
}

func (d *differ) compareSlices(expected, actual reflect.Value, path string) error {
	if expected.Len() != actual.Len() {
		return diffError(expected.Interface(), actual.Interface(), path)
	}

	for i := range expected.Len() {
		newPath := fmt.Sprintf("%s[%d]", path, i)
		if err := d.compare(expected.Index(i).Interface(), actual.Index(i).Interface(), newPath); err != nil {
			return err
		}
	}

	return nil
}

func compareStrings(expected, actual, path string) error {
	if strings.Contains(expected, "%") {
		// It is a wildcard string, convert to a regexp
		re, err := regexp.Compile("^" + toRegexp(expected) + "$")
		if err != nil {
			return fmt.Errorf("cannot compile wildcard expression %#v: %w", expected, err)
		}
		if !re.MatchString(actual) {
			return diffError(expected, actual, path)
		}
	} else if expected != actual {
		return diffError(expected, actual, path)
	}
	return nil
}

func diffError(expected, actual any, path string) error {
	expectedStr := formatValue(expected)
	actualStr := formatValue(actual)

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(expectedStr),
		B:        difflib.SplitLines(actualStr),
		FromFile: "expected",
		ToFile:   "actual",
		Context:  3,
	}
	diffStr, _ := difflib.GetUnifiedDiffString(diff)

	return fmt.Errorf("mismatch at path \"%s\":\n%s", path, diffStr)
}

func formatValue(v any) string {
	if v == nil {
		return "null"
	}
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(bytes)
}

// AssertJSON compares two JSON strings and allows using wildcards in expected value.
func AssertJSON(t assert.TestingT, expected string, actual string, msgAndArgs ...any) bool {
	err := CompareJSON(expected, actual)
	if err != nil {
		assert.Fail(t, err.Error(), msgAndArgs...)
		return false
	}
	return true
}

// isNumber checks if the value is a number.
func isNumber(_ string, actual any) bool {
	// JSON numbers are decoded as float64
	_, ok := actual.(float64)
	return ok
}

// isString checks if the value is a string.
func isString(expected string, actual any) bool {
	_, ok := actual.(string)
	if expected == "%S" && actual == nil {
		return true
	}
	return ok
}

// isDateTime checks if the value is a date-time string in RFC3339 format.
func isDateTime(_ string, actual any) bool {
	if str, ok := actual.(string); ok {
		_, err := time.Parse(time.RFC3339, str)
		return err == nil
	}
	return false
}

// isDate checks if the value is a date string in RFC3339 format.
func isDate(_ string, actual any) bool {
	if str, ok := actual.(string); ok {
		_, err := time.Parse("2006-01-02", str)
		return err == nil
	}
	return false
}

// stringify converts a value to its string representation for comparison.
func stringify(v any) string {
	if v == nil {
		return "null"
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		// Format float without decimal point if it's a whole number
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(val)
	default:
		// For other types, fallback to a simple format
		return fmt.Sprintf("%v", v)
	}
}

// toRegexp converts string with wildcards to regexp.
// It is a fixed version of wildcards.ToRegexp.
func toRegexp(input string) string {
	var out strings.Builder
	for len(input) > 0 {
		// Find next wildcard
		idx := strings.Index(input, "%")
		if idx == -1 {
			// No wildcard
			out.WriteString(regexp.QuoteMeta(input))
			break
		}

		// Quote part before wildcard
		out.WriteString(regexp.QuoteMeta(input[0:idx]))
		input = input[idx:]

		// Get wildcard
		if len(input) < 2 {
			// Unfinished wildcard
			out.WriteString(regexp.QuoteMeta(input))
			break
		}
		wildcard := input[0:2]
		input = input[2:]

		// Inspired by PhpUnit "assertStringMatchesFormat"
		// https://phpunit.readthedocs.io/en/9.5/assertions.html#assertstringmatchesformat
		switch wildcard {
		// %e: Represents a directory separator, for example / on Linux.
		case `%e`:
			out.WriteString(regexp.QuoteMeta(string(os.PathSeparator))) // nolint forbidigo
		// %s: One or more of anything (character or white space) except the end of line character.
		case `%s`:
			out.WriteString(`.+`)
		// %S: Zero or more of anything (character or white space) except the end of line character.
		case `%S`:
			out.WriteString(`.*`)
		// %a: One or more of anything (character or white space) including the end of line character.
		case `%a`:
			out.WriteString(`(.|\n)+`)
		// %A: Zero or more of anything (character or white space) including the end of line character.
		case `%A`:
			out.WriteString(`(.|\n)*`)
		// %w: Zero or more white space characters.
		case `%w`:
			out.WriteString(`\s*`)
		// %i: A signed integer value, for example +3142, -3142.
		case `%i`:
			out.WriteString(`[+-]?\d+`)
		// %d: An unsigned integer value, for example 123456.
		case `%d`:
			out.WriteString(`\d+`)
		// %x: One or more hexadecimal character. That is, characters in the range 0-9, a-f, A-F.
		case `%x`:
			out.WriteString(`[0-9a-fA-F]+`)
		// %f: A floating point number, for example: 3.142, -3.142, 3.142E-10, 3.142e+10.
		case `%f`:
			out.WriteString(`[-+]?[0-9]*\.?[0-9]+([eE][-+]?[0-9]+)?`)
		// %c: A single character of any sort.
		case `%c`:
			out.WriteString(`.`)
		// %%: A literal percent character: %.
		case `%%`:
			out.WriteString(`%`)
		default:
			out.WriteString(regexp.QuoteMeta(wildcard))
		}
	}
	return out.String()
}
