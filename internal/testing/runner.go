package testing

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner executes tests and returns structured results.
type Runner struct {
	Language   string
	ProjectDir string
	TmpDir     string
}

// Run executes the test suite and returns structured results.
func (r *Runner) Run(args []string) (*TestResult, error) {
	tmpDir, err := os.MkdirTemp("", "squire-test-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	xmlPath := filepath.Join(tmpDir, "results.xml")

	switch r.Language {
	case "Go":
		return r.runGo(xmlPath, args)
	case "TypeScript":
		return r.runTypeScript(xmlPath, args)
	case "Python":
		return r.runPython(xmlPath, args)
	case "C#":
		return r.runCSharp(xmlPath, args)
	default:
		return nil, fmt.Errorf("unsupported language: %s", r.Language)
	}
}

func (r *Runner) runGo(xmlPath string, args []string) (*TestResult, error) {
	// Prefer gotestsum if available (native JUnit XML support)
	if goTestSum, err := exec.LookPath("gotestsum"); err == nil {
		cmdArgs := []string{"--junitfile", xmlPath, "--"}
		cmdArgs = append(cmdArgs, defaultGoTestArgs(args)...)
		cmd := exec.Command(goTestSum, cmdArgs...)
		cmd.Dir = r.ProjectDir
		cmd.Stdout = os.Stderr // show progress on stderr
		cmd.Stderr = os.Stderr
		cmd.Run() // don't fail on test failures — we parse the XML

		return ParseJUnitXML(xmlPath)
	}

	// Fallback: go test with JSON output, convert to structured result
	cmdArgs := []string{"test", "-json"}
	cmdArgs = append(cmdArgs, defaultGoTestArgs(args)...)
	cmd := exec.Command("go", cmdArgs...)
	cmd.Dir = r.ProjectDir
	out, _ := cmd.Output() // don't fail on test failures

	return parseGoTestJSON(string(out))
}

func defaultGoTestArgs(args []string) []string {
	if len(args) > 0 {
		return args
	}
	return []string{"./..."}
}

func (r *Runner) runTypeScript(xmlPath string, args []string) (*TestResult, error) {
	// Jest with junit reporter
	cmd := exec.Command("npx", "jest", "--reporters=jest-junit")
	cmd.Dir = r.ProjectDir
	cmd.Env = append(os.Environ(), "JEST_JUNIT_OUTPUT_DIR="+filepath.Dir(xmlPath), "JEST_JUNIT_OUTPUT_NAME="+filepath.Base(xmlPath))
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Run()

	if _, err := os.Stat(xmlPath); err == nil {
		return ParseJUnitXML(xmlPath)
	}

	return nil, fmt.Errorf("jest-junit reporter not available. Install: npm install --save-dev jest-junit")
}

func (r *Runner) runPython(xmlPath string, args []string) (*TestResult, error) {
	cmdArgs := []string{"-m", "pytest", "--junitxml=" + xmlPath}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command("python3", cmdArgs...)
	cmd.Dir = r.ProjectDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Run()

	if _, err := os.Stat(xmlPath); err == nil {
		return ParseJUnitXML(xmlPath)
	}

	return nil, fmt.Errorf("pytest not available. Install: pip install pytest")
}

func (r *Runner) runCSharp(xmlPath string, args []string) (*TestResult, error) {
	cmdArgs := []string{"test", "--logger", "trx;LogFileName=" + xmlPath}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command("dotnet", cmdArgs...)
	cmd.Dir = r.ProjectDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Run()

	// .NET produces TRX format, not JUnit — convert or parse directly
	// For now, try JUnit logger
	if _, err := os.Stat(xmlPath); err == nil {
		return ParseJUnitXML(xmlPath)
	}

	return nil, fmt.Errorf("dotnet test completed but no JUnit XML output found")
}

// parseGoTestJSON parses `go test -json` output into a TestResult.
// This is the fallback when gotestsum is not installed.
func parseGoTestJSON(output string) (*TestResult, error) {
	result := &TestResult{}
	testResults := map[string]string{} // "pkg/TestName" → "pass"|"fail"|"skip"
	testOutput := map[string][]string{} // "pkg/TestName" → output lines

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] != '{' {
			continue
		}

		// Quick JSON parsing without encoding/json for speed
		action := extractJSONField(line, "Action")
		test := extractJSONField(line, "Test")
		pkg := extractJSONField(line, "Package")
		outputLine := extractJSONField(line, "Output")

		key := pkg + "/" + test
		if test == "" {
			key = pkg + "/"
		}

		switch action {
		case "pass":
			if test != "" {
				testResults[key] = "pass"
				result.Passed++
			}
		case "fail":
			if test != "" {
				testResults[key] = "fail"
				result.Failed++
				result.Failures = append(result.Failures, TestFailure{
					TestName: test,
					Package:  pkg,
					Detail:   strings.Join(testOutput[key], ""),
				})
			}
		case "skip":
			if test != "" {
				testResults[key] = "skip"
				result.Skipped++
			}
		case "output":
			if test != "" {
				testOutput[key] = append(testOutput[key], outputLine)
			}
		}
	}

	result.Total = result.Passed + result.Failed + result.Skipped

	// Extract source refs from failure output
	for i := range result.Failures {
		result.Failures[i].SourceRef = extractSourceRef(result.Failures[i].Detail, "")
		result.Failures[i].Message = extractFirstAssertionLine(result.Failures[i].Detail)
	}

	return result, nil
}

func extractJSONField(line, field string) string {
	key := `"` + field + `":`
	idx := strings.Index(line, key)
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(line[idx+len(key):])
	if len(rest) == 0 {
		return ""
	}
	if rest[0] == '"' {
		end := strings.Index(rest[1:], `"`)
		if end < 0 {
			return ""
		}
		return rest[1 : end+1]
	}
	// Non-string value
	end := strings.IndexAny(rest, ",}")
	if end < 0 {
		return rest
	}
	return strings.TrimSpace(rest[:end])
}

func extractFirstAssertionLine(detail string) string {
	// Clean up escaped newlines from JSON output
	detail = strings.ReplaceAll(detail, "\\n", "\n")

	for _, line := range strings.Split(detail, "\n") {
		line = strings.TrimSpace(line)
		// Skip blank lines and test framework boilerplate
		if line == "" || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "===") || strings.HasPrefix(line, "=== RUN") {
			continue
		}
		// Skip PASS/FAIL lines
		if strings.HasPrefix(line, "FAIL") || strings.HasPrefix(line, "PASS") || strings.HasPrefix(line, "ok ") {
			continue
		}
		// Lines with file:line references that contain the assertion message
		if strings.Contains(line, "_test.go:") {
			// Extract just the message part after "file_test.go:42: message"
			if idx := strings.Index(line, ": "); idx > 0 {
				msg := strings.TrimSpace(line[idx+2:])
				if msg != "" {
					return msg
				}
			}
			return line
		}
		// Direct assertion messages
		if strings.Contains(line, "expected") || strings.Contains(line, "got") ||
			strings.Contains(line, "want") || strings.Contains(line, "assert") ||
			strings.Contains(line, "Error") {
			return line
		}
	}
	return ""
}
