// Package testing provides structured test execution and JUnit XML parsing.
package testing

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

// JUnitTestSuites is the top-level XML element.
type JUnitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a single test suite (package).
type JUnitTestSuite struct {
	XMLName  xml.Name        `xml:"testsuite"`
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Errors   int             `xml:"errors,attr"`
	Skipped  int             `xml:"skipped,attr"`
	Time     string          `xml:"time,attr"`
	Cases    []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a single test.
type JUnitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
	Error     *JUnitError   `xml:"error,omitempty"`
	Skipped   *JUnitSkipped `xml:"skipped,omitempty"`
}

// JUnitFailure is a test assertion failure.
type JUnitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

// JUnitError is a test error (panic, unexpected).
type JUnitError struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

// JUnitSkipped marks a skipped test.
type JUnitSkipped struct {
	Message string `xml:"message,attr"`
}

// TestResult is the structured output from a test run.
type TestResult struct {
	Total    int
	Passed   int
	Failed   int
	Skipped  int
	Errors   int
	Duration string
	Failures []TestFailure
}

// TestFailure describes a single test failure with context.
type TestFailure struct {
	TestName  string
	Package   string
	Duration  string
	Message   string // assertion message
	Detail    string // full failure output
	SourceRef string // file:line if extractable
}

// ParseJUnitXML reads and parses a JUnit XML results file.
func ParseJUnitXML(path string) (*TestResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading results: %w", err)
	}

	// Try testsuites wrapper first
	var suites JUnitTestSuites
	if err := xml.Unmarshal(data, &suites); err != nil || len(suites.Suites) == 0 {
		// Try single testsuite
		var suite JUnitTestSuite
		if err := xml.Unmarshal(data, &suite); err != nil {
			return nil, fmt.Errorf("parsing JUnit XML: %w", err)
		}
		suites.Suites = []JUnitTestSuite{suite}
	}

	result := &TestResult{}

	for _, suite := range suites.Suites {
		result.Total += suite.Tests
		result.Failed += suite.Failures
		result.Errors += suite.Errors
		result.Skipped += suite.Skipped
		if suite.Time != "" {
			result.Duration = suite.Time
		}

		for _, tc := range suite.Cases {
			if tc.Failure != nil {
				f := TestFailure{
					TestName: tc.Name,
					Package:  suite.Name,
					Duration: tc.Time,
					Message:  tc.Failure.Message,
					Detail:   tc.Failure.Body,
				}
				f.SourceRef = extractSourceRef(tc.Failure.Body, tc.Classname)
				result.Failures = append(result.Failures, f)
			}
			if tc.Error != nil {
				f := TestFailure{
					TestName: tc.Name,
					Package:  suite.Name,
					Duration: tc.Time,
					Message:  tc.Error.Message,
					Detail:   tc.Error.Body,
				}
				f.SourceRef = extractSourceRef(tc.Error.Body, tc.Classname)
				result.Failures = append(result.Failures, f)
			}
		}
	}

	result.Passed = result.Total - result.Failed - result.Errors - result.Skipped
	if result.Passed < 0 {
		result.Passed = 0
	}

	return result, nil
}

// extractSourceRef tries to find a file:line reference in failure output.
func extractSourceRef(body, classname string) string {
	body = strings.ReplaceAll(body, "\\n", "\n")
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		// Look for patterns like "file.go:123:" or "file_test.go:42"
		if idx := strings.Index(line, "_test.go:"); idx >= 0 {
			end := idx + 9 // past "_test.go:"
			for end < len(line) && line[end] >= '0' && line[end] <= '9' {
				end++
			}
			return line[:end]
		}
		if idx := strings.Index(line, ".go:"); idx >= 0 {
			end := idx + 4 // past ".go:"
			for end < len(line) && line[end] >= '0' && line[end] <= '9' {
				end++
			}
			return line[:end]
		}
	}
	return ""
}
