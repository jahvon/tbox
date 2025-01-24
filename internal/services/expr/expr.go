package expr

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/expr-lang/expr"

	"github.com/jahvon/flow/internal/context"
	"github.com/jahvon/flow/types/executable"
)

func IsTruthy(ex string, env any) (bool, error) {
	output, err := Evaluate(ex, env)
	if err != nil {
		return false, err
	}

	switch v := output.(type) {
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
		return false, nil
	}
}

func Evaluate(ex string, env any) (interface{}, error) {
	program, err := expr.Compile(ex, expr.Env(env))
	if err != nil {
		return nil, err
	}

	output, err := expr.Run(program, env)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func EvaluateString(ex string, env any) (string, error) {
	output, err := Evaluate(ex, env)
	if err != nil {
		return "", err
	}
	str, ok := output.(string)
	if !ok {
		return "", fmt.Errorf("expected string, got %T", output)
	}
	return str, nil
}

type CtxData struct {
	Workspace     string `expr:"workspace"`
	Namespace     string `expr:"namespace"`
	WorkspacePath string `expr:"workspacePath"`
	FlowFileName  string `expr:"flowFileName"`
	FlowFilePath  string `expr:"flowFilePath"`
	FlowFileDir   string `expr:"flowFileDir"`
}

type ExpressionData struct {
	OS    string            `expr:"os"`
	Arch  string            `expr:"arch"`
	Ctx   *CtxData          `expr:"ctx"`
	Store map[string]string `expr:"store"`
	Env   map[string]string `expr:"env"`
}

func ExpressionEnv(
	ctx *context.Context,
	executable *executable.Executable,
	dataMap, envMap map[string]string,
) ExpressionData {
	fn := filepath.Base(filepath.Base(executable.FlowFilePath()))
	return ExpressionData{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
		Ctx: &CtxData{
			Workspace:     ctx.CurrentWorkspace.AssignedName(),
			Namespace:     ctx.Config.CurrentNamespace,
			WorkspacePath: executable.WorkspacePath(),
			FlowFileName:  fn,
			FlowFilePath:  executable.FlowFilePath(),
			FlowFileDir:   filepath.Dir(executable.FlowFilePath()),
		},
		Store: dataMap,
		Env:   envMap,
	}
}
