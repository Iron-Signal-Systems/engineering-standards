package secrets

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	approvedExternalReferencePattern = regexp.MustCompile(
		`(?i)\b(?:secret|vault|keyring|credential)://[^\s"'<>]+`,
	)
	goCommentReferencePattern = regexp.MustCompile(
		`(?i)^(?:[A-Za-z_][A-Za-z0-9_]*\.)*[A-Za-z_][A-Za-z0-9_]*(?:secret|token|password|passwd|key|credential)[A-Za-z0-9_]*$`,
	)
)

type byteRange struct {
	Start int
	End   int
}

type semanticContext struct {
	ExternalReferences []byteRange
	GoExpressions      []byteRange
	GoComments         []byteRange
}

func semanticContextFor(path string, data []byte) semanticContext {
	context := semanticContext{
		ExternalReferences: matchRanges(approvedExternalReferencePattern, data),
	}
	context.GoExpressions, context.GoComments = goSemanticRanges(path, data)
	return context
}

func ignoreSensitiveAssignment(
	path string,
	data []byte,
	fullStart, fullEnd, valueStart, valueEnd int,
	semantics semanticContext,
) bool {
	if rangeContained(semantics.ExternalReferences, fullStart, fullEnd) {
		return true
	}
	if rangeStartsWithin(semantics.GoExpressions, valueStart) {
		return true
	}
	if rangeStartsWithin(semantics.GoComments, fullStart) &&
		goCommentReferencePattern.Match(trimReferenceValue(data[valueStart:valueEnd])) {
		return true
	}
	return shellDynamicAssignment(path, data, valueStart)
}

func matchRanges(pattern *regexp.Regexp, data []byte) []byteRange {
	matches := pattern.FindAllIndex(data, -1)
	ranges := make([]byteRange, 0, len(matches))
	for _, match := range matches {
		ranges = append(ranges, byteRange{Start: match[0], End: match[1]})
	}
	return ranges
}

func rangeContained(ranges []byteRange, start, end int) bool {
	for _, current := range ranges {
		if start >= current.Start && end <= current.End {
			return true
		}
	}
	return false
}

func rangeStartsWithin(ranges []byteRange, offset int) bool {
	for _, current := range ranges {
		if offset >= current.Start && offset < current.End {
			return true
		}
	}
	return false
}

func trimReferenceValue(value []byte) []byte {
	return bytes.TrimRight(value, "}],);\r\n\t ")
}

func goSemanticRanges(path string, data []byte) ([]byteRange, []byteRange) {
	if strings.ToLower(filepath.Ext(path)) != ".go" {
		return nil, nil
	}

	files := token.NewFileSet()
	file, err := parser.ParseFile(files, path, data, parser.AllErrors|parser.ParseComments)
	if err != nil || file == nil {
		// Malformed Go-like source remains fail-closed and receives ordinary
		// text classification instead of inferred Go semantics.
		return nil, nil
	}

	var expressions []byteRange
	addExpression := func(expression ast.Expr) {
		if !simpleReferenceExpression(expression) {
			return
		}
		start := files.PositionFor(expression.Pos(), false).Offset
		end := files.PositionFor(expression.End(), false).Offset
		if start >= 0 && end > start {
			expressions = append(expressions, byteRange{Start: start, End: end})
		}
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch current := node.(type) {
		case *ast.KeyValueExpr:
			addExpression(current.Value)
		case *ast.AssignStmt:
			for _, expression := range current.Rhs {
				addExpression(expression)
			}
		case *ast.ValueSpec:
			for _, expression := range current.Values {
				addExpression(expression)
			}
		}
		return true
	})

	comments := make([]byteRange, 0, len(file.Comments))
	for _, group := range file.Comments {
		start := files.PositionFor(group.Pos(), false).Offset
		end := files.PositionFor(group.End(), false).Offset
		if start >= 0 && end > start {
			comments = append(comments, byteRange{Start: start, End: end})
		}
	}

	return expressions, comments
}

func simpleReferenceExpression(expression ast.Expr) bool {
	switch current := expression.(type) {
	case *ast.Ident:
		return true
	case *ast.SelectorExpr:
		return simpleReferenceExpression(current.X)
	default:
		return false
	}
}

func shellDynamicAssignment(path string, data []byte, start int) bool {
	if !shellSource(path, data) || start < 0 || start >= len(data) {
		return false
	}

	end := len(data)
	if offset := bytes.IndexByte(data[start:], '\n'); offset >= 0 {
		end = start + offset
	}
	value := strings.TrimSpace(string(data[start:end]))
	value = strings.TrimLeft(value, `"'`)

	return strings.HasPrefix(value, "$(") ||
		strings.HasPrefix(value, "${") ||
		strings.HasPrefix(value, "$((") ||
		strings.HasPrefix(value, "$") ||
		strings.HasPrefix(value, "`") ||
		(strings.HasPrefix(value, "(") && strings.ContainsAny(value, "$`"))
}

func shellSource(path string, data []byte) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".sh", ".bash", ".zsh", ".ksh":
		return true
	}

	firstLine := data
	if offset := bytes.IndexByte(data, '\n'); offset >= 0 {
		firstLine = data[:offset]
	}
	return bytes.HasPrefix(firstLine, []byte("#!")) &&
		strings.Contains(strings.ToLower(string(firstLine)), "sh")
}
