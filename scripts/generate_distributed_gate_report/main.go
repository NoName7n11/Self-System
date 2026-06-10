package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

type goTestEvent struct {
	Time    string  `json:"Time"`
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
	Output  string  `json:"Output"`
}

type testOutcome struct {
	Package string
	Test    string
	Status  string
	Elapsed float64
}

func main() {
	inputPath := flag.String("input", "", "Path to go test -json output")
	outputPath := flag.String("output", "", "Path to output markdown report")
	command := flag.String("command", `go test ./internal/sync ./test/integration -run "Sync|Offline|Replay"`, "Command used for gate execution")
	failOnFailures := flag.Bool("fail-on-failures", false, "Exit non-zero if failed tests are detected")
	flag.Parse()

	if strings.TrimSpace(*inputPath) == "" {
		fatal("input path is required")
	}
	if strings.TrimSpace(*outputPath) == "" {
		fatal("output path is required")
	}

	report, hasFailures, err := buildReport(*inputPath, strings.TrimSpace(*command))
	if err != nil {
		fatal(err.Error())
	}

	if err := os.WriteFile(*outputPath, []byte(report), 0o644); err != nil {
		fatal(fmt.Sprintf("write report: %v", err))
	}

	if hasFailures && *failOnFailures {
		os.Exit(1)
	}
}

func buildReport(inputPath, command string) (string, bool, error) {
	file, err := os.Open(inputPath)
	if err != nil {
		return "", false, fmt.Errorf("open input: %w", err)
	}
	defer file.Close()

	testOutcomes := map[string]testOutcome{}
	packageStatus := map[string]string{}
	packageElapsed := map[string]float64{}
	packageErrors := map[string][]string{}

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event goTestEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return "", false, fmt.Errorf("decode event at line %d: %w", lineNumber, err)
		}

		pkg := strings.TrimSpace(event.Package)
		testName := strings.TrimSpace(event.Test)
		action := strings.TrimSpace(event.Action)

		if pkg != "" && action == "output" && testName == "" {
			outputLine := strings.TrimSpace(event.Output)
			if strings.HasPrefix(outputLine, "FAIL") || strings.Contains(strings.ToLower(outputLine), "panic") {
				packageErrors[pkg] = append(packageErrors[pkg], outputLine)
			}
		}

		if testName != "" {
			key := pkg + "::" + testName
			outcome := testOutcomes[key]
			outcome.Package = pkg
			outcome.Test = testName
			if action == "pass" || action == "fail" || action == "skip" {
				outcome.Status = action
				outcome.Elapsed = event.Elapsed
				testOutcomes[key] = outcome
			} else if _, exists := testOutcomes[key]; !exists {
				outcome.Status = "run"
				testOutcomes[key] = outcome
			}
			continue
		}

		if pkg != "" && (action == "pass" || action == "fail") {
			packageStatus[pkg] = action
			packageElapsed[pkg] = event.Elapsed
		}
	}

	if err := scanner.Err(); err != nil {
		return "", false, fmt.Errorf("scan input: %w", err)
	}

	passed := 0
	failed := 0
	skipped := 0
	failedTests := make([]testOutcome, 0)

	for _, outcome := range testOutcomes {
		switch outcome.Status {
		case "pass":
			passed++
		case "fail":
			failed++
			failedTests = append(failedTests, outcome)
		case "skip":
			skipped++
		}
	}

	packageNames := make([]string, 0, len(packageStatus))
	for pkg := range packageStatus {
		packageNames = append(packageNames, pkg)
	}
	sort.Strings(packageNames)
	if len(packageNames) == 0 {
		for pkg := range packageErrors {
			packageNames = append(packageNames, pkg)
		}
		sort.Strings(packageNames)
	}

	overallPass := failed == 0
	for _, status := range packageStatus {
		if status == "fail" {
			overallPass = false
			break
		}
	}

	var builder strings.Builder
	builder.WriteString("# Distributed Behavior Gate Report\n\n")
	builder.WriteString(fmt.Sprintf("Generated: %s UTC\n\n", time.Now().UTC().Format(time.RFC3339)))
	builder.WriteString("## Command\n\n")
	builder.WriteString("```bash\n")
	builder.WriteString(command)
	builder.WriteString("\n```\n\n")
	builder.WriteString("## Result\n\n")
	if overallPass {
		builder.WriteString("- Overall status: PASS\n")
	} else {
		builder.WriteString("- Overall status: FAIL\n")
	}
	builder.WriteString(fmt.Sprintf("- Tests passed: %d\n", passed))
	builder.WriteString(fmt.Sprintf("- Tests failed: %d\n", failed))
	builder.WriteString(fmt.Sprintf("- Tests skipped: %d\n", skipped))
	builder.WriteString("\n")

	builder.WriteString("## Package Outcomes\n\n")
	if len(packageNames) == 0 {
		builder.WriteString("- No package-level status events captured.\n")
	} else {
		for _, pkg := range packageNames {
			status := packageStatus[pkg]
			if status == "" {
				status = "unknown"
			}
			elapsed := packageElapsed[pkg]
			builder.WriteString(fmt.Sprintf("- %s: %s (elapsed %.2fs)\n", pkg, strings.ToUpper(status), elapsed))
			if lines := packageErrors[pkg]; len(lines) > 0 {
				builder.WriteString(fmt.Sprintf("  - error excerpt: %s\n", lines[0]))
			}
		}
	}
	builder.WriteString("\n")

	builder.WriteString("## Failed Tests\n\n")
	if len(failedTests) == 0 {
		builder.WriteString("- None\n")
	} else {
		sort.Slice(failedTests, func(i, j int) bool {
			if failedTests[i].Package == failedTests[j].Package {
				return failedTests[i].Test < failedTests[j].Test
			}
			return failedTests[i].Package < failedTests[j].Package
		})
		for _, outcome := range failedTests {
			builder.WriteString(fmt.Sprintf("- %s::%s (%.2fs)\n", outcome.Package, outcome.Test, outcome.Elapsed))
		}
	}

	return builder.String(), !overallPass, nil
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(2)
}
