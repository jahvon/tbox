package expr

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/template"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// Template wraps text/template but evaluates expressions using expr instead
type Template struct {
	name      string
	text      string
	data      any
	tmpl      *template.Template
	exprCache map[string]*vm.Program
}

func NewTemplate(name string, data any) *Template {
	t := &Template{
		name:      name,
		data:      data,
		exprCache: make(map[string]*vm.Program),
	}
	return t
}

func (t *Template) Parse(text string) error {
	t.text = text
	processed := t.preProcessExpressions(text)
	tmpl := template.New(t.name).Funcs(template.FuncMap{"expr": t.evalExpr, "exprBool": t.evalExprBool})

	parsed, err := tmpl.Parse(processed)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	t.tmpl = parsed
	return nil
}

func (t *Template) Execute(wr io.Writer) error {
	if t.tmpl == nil {
		return fmt.Errorf("template not parsed")
	}

	return t.tmpl.Execute(wr, t.data)
}

func (t *Template) ExecuteToString() (string, error) {
	var buf bytes.Buffer
	err := t.Execute(&buf)
	return buf.String(), err
}

func (t *Template) compileExpr(expression string) (*vm.Program, error) {
	if node, ok := t.exprCache[expression]; ok {
		return node, nil
	}

	compiled, err := expr.Compile(expression, expr.Env(t.data))
	if err != nil {
		return nil, err
	}
	t.exprCache[expression] = compiled
	return compiled, nil
}

//nolint:funlen
func (t *Template) preProcessExpressions(text string) string {
	var result strings.Builder
	remaining := text
	contextDepth := 0 // Track nested range/with blocks

	for {
		start := strings.Index(remaining, "{{")
		if start == -1 {
			result.WriteString(remaining)
			break
		}
		result.WriteString(remaining[:start])

		end := strings.Index(remaining[start:], "}}")
		if end == -1 {
			result.WriteString(remaining[start:])
			break
		}
		end += start

		action := remaining[start+2 : end]
		trimLeft := strings.HasPrefix(action, "-")
		trimRight := strings.HasSuffix(action, "-")
		action = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(action, "-"), "-"))

		result.WriteString("{{")
		if trimLeft {
			result.WriteString("-")
		}
		result.WriteString(" ")

		switch {
		case strings.HasPrefix(action, "if "):
			condition := strings.TrimPrefix(action, "if ")
			result.WriteString("if exprBool `")
			result.WriteString(strings.TrimSpace(condition))
			result.WriteString("`")
		case strings.HasPrefix(action, "with "):
			value := strings.TrimPrefix(action, "with ")
			result.WriteString("with expr `")
			result.WriteString(strings.TrimSpace(value))
			result.WriteString("`")
			contextDepth++
		case action == "end":
			result.WriteString("end")
			if contextDepth > 0 {
				contextDepth--
			}
		case action == "else":
			result.WriteString("else")
		case strings.HasPrefix(action, "range "):
			value := strings.TrimPrefix(action, "range ")
			result.WriteString("range expr `")
			result.WriteString(strings.TrimSpace(value))
			result.WriteString("`")
			contextDepth++
		default:
			if contextDepth > 0 && (strings.HasPrefix(action, ".") || action == ".") {
				result.WriteString(action)
			} else {
				result.WriteString("expr `")
				result.WriteString(strings.TrimSpace(action))
				result.WriteString("`")
			}
		}

		result.WriteString(" ")
		if trimRight {
			result.WriteString("-")
		}
		result.WriteString("}}")

		remaining = remaining[end+2:]
	}

	return result.String()
}

func (t *Template) evalExpr(expression string) (interface{}, error) {
	program, err := t.compileExpr(expression)
	if err != nil {
		return nil, fmt.Errorf("compiling expression: %w", err)
	}
	result, err := expr.Run(program, t.data)
	if err != nil {
		return nil, fmt.Errorf("evaluating expression: %w", err)
	}

	return result, nil
}

func (t *Template) evalExprBool(expression string) (bool, error) {
	result, err := t.evalExpr(expression)
	if err != nil {
		return false, err
	}

	switch v := result.(type) {
	case bool:
		return v, nil
	case int, int64, float64, uint, uint64:
		return v != 0, nil
	case string:
		truthy, err := strconv.ParseBool(strings.Trim(v, `"' `))
		if err != nil {
			return false, err
		}
		return truthy, nil
	default:
		return result != nil, nil
	}
}
