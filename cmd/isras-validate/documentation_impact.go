package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/docimpact"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/redact"
	"github.com/Iron-Signal-Systems/engineering-standards/internal/validation"
)

type documentationImpactOptions struct {
	BaseCommit string
	HeadCommit string
}

func runDocumentationImpact(
	ctx context.Context,
	runner *validation.Runner,
	args []string,
) int {
	options, err := parseDocumentationImpactOptions(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		return 2
	}

	result, executeErr := docimpact.Run(
		ctx,
		docimpact.Request{
			Root:       runner.Root,
			BaseCommit: options.BaseCommit,
			HeadCommit: options.HeadCommit,
		},
	)
	renderDocumentationImpactResult(result, runner.Root)
	if executeErr != nil {
		fmt.Fprintln(
			os.Stderr,
			"FAIL:",
			redact.Sanitize(executeErr.Error()),
		)
		return 1
	}
	return 0
}

func parseDocumentationImpactOptions(
	args []string,
) (documentationImpactOptions, error) {
	var options documentationImpactOptions
	baseSeen := false
	headSeen := false

	for index := 0; index < len(args); index++ {
		argument := args[index]
		switch {
		case argument == "--base":
			if baseSeen {
				return options, errors.New(
					"--base may be declared only once",
				)
			}
			if index+1 >= len(args) {
				return options, errors.New(
					"--base requires an exact commit ID",
				)
			}
			index++
			options.BaseCommit = args[index]
			baseSeen = true
		case strings.HasPrefix(argument, "--base="):
			if baseSeen {
				return options, errors.New(
					"--base may be declared only once",
				)
			}
			options.BaseCommit = strings.TrimPrefix(
				argument,
				"--base=",
			)
			baseSeen = true
		case argument == "--head":
			if headSeen {
				return options, errors.New(
					"--head may be declared only once",
				)
			}
			if index+1 >= len(args) {
				return options, errors.New(
					"--head requires an exact commit ID",
				)
			}
			index++
			options.HeadCommit = args[index]
			headSeen = true
		case strings.HasPrefix(argument, "--head="):
			if headSeen {
				return options, errors.New(
					"--head may be declared only once",
				)
			}
			options.HeadCommit = strings.TrimPrefix(
				argument,
				"--head=",
			)
			headSeen = true
		default:
			return options, fmt.Errorf(
				"documentation-impact does not accept %q",
				argument,
			)
		}
	}
	if !baseSeen || options.BaseCommit == "" {
		return options, errors.New(
			"--base is required",
		)
	}
	if !headSeen || options.HeadCommit == "" {
		return options, errors.New(
			"--head is required",
		)
	}
	return options, nil
}

func renderDocumentationImpactResult(
	result docimpact.Result,
	root string,
) {
	fmt.Println("DOCUMENTATION IMPACT VALIDATION")
	fmt.Println("===============================")
	fmt.Printf("Status:          %s\n", result.Evidence.Status)
	fmt.Printf("Policy:          %s\n", result.Evidence.Policy.Path)
	fmt.Printf("Policy SHA-256:  %s\n", result.Evidence.Policy.SHA256)
	fmt.Printf("Requested base:  %s\n", result.Evidence.Comparison.RequestedBase)
	fmt.Printf("Requested head:  %s\n", result.Evidence.Comparison.RequestedHead)
	fmt.Printf("Resolved base:   %s\n", result.Evidence.Comparison.BaseCommit)
	fmt.Printf("Resolved head:   %s\n", result.Evidence.Comparison.HeadCommit)
	fmt.Printf("Merge base:      %s\n", result.Evidence.Comparison.MergeBase)
	fmt.Printf(
		"Changed paths:   %d\n",
		len(result.Evidence.Comparison.ChangedPaths),
	)
	fmt.Printf(
		"Triggered rules: %d\n",
		len(result.Evidence.Report.Triggered),
	)
	for _, rule := range result.Evidence.Report.Triggered {
		fmt.Printf("  %s: %s\n", rule.ID, rule.Status)
		for _, requirement := range rule.Requirements {
			fmt.Printf(
				"    %s: %s\n",
				requirement.ID,
				requirement.Status,
			)
		}
	}
	if result.Evidence.Failure != "" {
		fmt.Printf(
			"Failure:         %s\n",
			redact.Sanitize(result.Evidence.Failure),
		)
	}
	if result.EvidenceJSON != "" {
		fmt.Printf(
			"JSON evidence:   %s\n",
			relativePath(root, result.EvidenceJSON),
		)
	}
	if result.EvidenceText != "" {
		fmt.Printf(
			"Text evidence:   %s\n",
			relativePath(root, result.EvidenceText),
		)
	}
}
