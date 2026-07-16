package dashboard

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/Iron-Signal-Systems/engineering-standards/internal/model"
)

const nameWidth = 30

type Printer struct {
	out   io.Writer
	color bool
}

func New(out io.Writer) *Printer {
	color := false
	if file, ok := out.(*os.File); ok && os.Getenv("NO_COLOR") == "" {
		if stat, err := file.Stat(); err == nil {
			color = stat.Mode()&os.ModeCharDevice != 0
		}
	}
	return &Printer{out: out, color: color}
}

func (p *Printer) Header(profile string) {
	fmt.Fprintln(p.out)
	fmt.Fprintf(p.out, "%s %s\n", p.paint("IRON SIGNAL · ISRAS VALIDATION", "36;1"), p.paint("["+profile+"]", "33;1"))
	fmt.Fprintln(p.out, p.paint("repository integrity · Go quality · secret protection", "2"))
	fmt.Fprintln(p.out, strings.Repeat("─", 68))
}

func (p *Printer) Checks(checks []model.Check) {
	sectionOrder := []string{"SYSTEM", "REPOSITORY", "GO SOURCE", "SECRET PROTECTION", "RESULT"}
	bySection := make(map[string][]model.Check)
	for _, check := range checks {
		bySection[check.Section] = append(bySection[check.Section], check)
	}
	seen := make(map[string]bool)
	for _, section := range sectionOrder {
		if len(bySection[section]) == 0 {
			continue
		}
		p.section(section, bySection[section])
		seen[section] = true
	}
	var extras []string
	for section := range bySection {
		if !seen[section] {
			extras = append(extras, section)
		}
	}
	sort.Strings(extras)
	for _, section := range extras {
		p.section(section, bySection[section])
	}
}

func (p *Printer) section(section string, checks []model.Check) {
	fmt.Fprintln(p.out)
	fmt.Fprintln(p.out, p.paint(section, "1"))
	for _, check := range checks {
		p.check(check)
	}
}

func (p *Printer) check(check model.Check) {
	marker, statusStyle := "◆", "36;1"
	switch check.Status {
	case model.Pass:
		marker, statusStyle = "●", "32;1"
	case model.Fail:
		marker, statusStyle = "●", "31;1"
	case model.Warn:
		marker, statusStyle = "▲", "33;1"
	}
	name := check.Name
	if len([]rune(name)) > nameWidth {
		name = string([]rune(name)[:nameWidth])
	}
	fmt.Fprintf(p.out, "%-30s %s %-5s  %s\n",
		p.paint(name, "36"),
		p.paint(marker, statusStyle),
		p.paint(string(check.Status), statusStyle),
		p.paint(check.Detail, "2"),
	)
	if check.LogPath != "" {
		fmt.Fprintf(p.out, "%-30s %s %-5s  %s\n", "Failure log", p.paint("◆", "36;1"), p.paint("INFO", "36;1"), check.LogPath)
	}
	if len(check.Actions) > 0 {
		fmt.Fprintln(p.out)
		fmt.Fprintln(p.out, p.paint("  AVAILABLE ACTIONS", "1"))
		for _, action := range check.Actions {
			fmt.Fprintf(p.out, "\n  %s\n", p.paint("["+action.Label+"]", "33;1"))
			if action.Description != "" {
				fmt.Fprintf(p.out, "  %s\n", action.Description)
			}
			for _, line := range strings.Split(action.Command, "\n") {
				fmt.Fprintf(p.out, "    %s\n", p.paint(line, "36"))
			}
		}
	}
}

func (p *Printer) Footer(summary model.Summary) {
	fmt.Fprintln(p.out)
	fmt.Fprintln(p.out, strings.Repeat("─", 68))
	if summary.Failed() {
		fmt.Fprintf(p.out, "%s Resolve the listed failures, then run:\n\n  %s\n\n",
			p.paint("Not ready.", "31;1"), p.paint("./.local/bin/isras-validate all", "36"))
		return
	}
	fmt.Fprintf(p.out, "%s The declared validation completed successfully.\n\n", p.paint("Ready.", "32;1"))
}

func (p *Printer) paint(value, code string) string {
	if !p.color {
		return value
	}
	return "\x1b[" + code + "m" + value + "\x1b[0m"
}
