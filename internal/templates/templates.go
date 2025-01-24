package templates

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"time"

	tuikitIO "github.com/jahvon/tuikit/io"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/jahvon/flow/internal/context"
	"github.com/jahvon/flow/internal/filesystem"
	"github.com/jahvon/flow/internal/runner"
	"github.com/jahvon/flow/internal/runner/engine"
	"github.com/jahvon/flow/internal/services/expr"
	"github.com/jahvon/flow/internal/utils"
	argUtils "github.com/jahvon/flow/internal/utils/args"
	execUtils "github.com/jahvon/flow/internal/utils/executables"
	"github.com/jahvon/flow/types/executable"
	"github.com/jahvon/flow/types/workspace"
)

func ProcessTemplate(
	ctx *context.Context,
	template *executable.Template,
	ws *workspace.Workspace,
	flowfileName, flowfileDir string,
) error {
	logger := ctx.Logger
	if flowfileName == "" {
		flowfileName = fmt.Sprintf("executables_%s", time.Now().Format("20060102150405"))
	}
	flowfileName = strings.ReplaceAll(strings.ToLower(flowfileName), " ", "_")
	if !strings.HasSuffix(flowfileName, executable.FlowFileExt) {
		flowfileName += executable.FlowFileExt
	}

	formMap := make(map[string]string)
	if template.Form != nil {
		if err := showForm(ctx, template.Form); err != nil {
			return err
		}
		formMap = template.Form.ValueMap()
	}

	env := os.Environ()
	envMap := make(map[string]string)
	for _, e := range env {
		pair := strings.SplitN(e, "=", 2)
		envMap[pair[0]] = pair[1]
	}
	flowfileDir = utils.ExpandDirectory(logger, flowfileDir, ws.Location(), template.Location(), envMap)
	fullPath := filepath.Join(flowfileDir, flowfileName)
	logger.Debugx(
		fmt.Sprintf("processing %s template", flowfileName),
		"template", template.Location(), "output", fullPath,
	)

	dataMap := newExpressionData(
		ws.AssignedName(), ws.Location(),
		flowfileName, flowfileDir, fullPath, template.Location(),
		envMap, formMap,
	)

	if err := runExecutables(
		ctx, ws, "pre-run", filepath.Dir(template.Location()), template.PreRun, dataMap,
	); err != nil {
		return err
	}
	if err := copyAllArtifacts(
		logger,
		template.Artifacts,
		ws.Location(),
		filepath.Dir(template.Location()),
		flowfileDir,
		dataMap,
	); err != nil {
		return err
	}

	if template.Template != "" {
		flowfile, err := templateToFlowfile(template, dataMap)
		if err != nil {
			return err
		}

		if _, e := os.Stat(fullPath); e == nil {
			// TODO: Add a flag to overwrite existing files
			logger.Warnx("Overwriting existing file", "dst", fullPath)
		}

		if err := filesystem.WriteFlowFile(fullPath, flowfile); err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to write flowfile %s from template", flowfileName))
		}
	}
	if err := runExecutables(ctx, ws, "post-run", flowfileDir, template.PostRun, dataMap); err != nil {
		return err
	}

	return nil
}

//nolint:gocognit
func runExecutables(
	ctx *context.Context,
	ws *workspace.Workspace,
	stage, flowfileDir string,
	execs []executable.TemplateRefConfig,
	templateData expressionData,
) error {
	ctx.Logger.Debugf("running %d %s executables", len(execs), stage)
	for i, e := range execs {
		if e.If != "" {
			eval, err := expr.IsTruthy(e.If, templateData)
			if err != nil {
				return errors.Wrap(err, "unable to evaluate if condition")
			}
			if !eval {
				ctx.Logger.Debugf("skipping %s executable %d", stage, i)
				return nil
			}
		}
		var exec *executable.Executable
		switch {
		case e.Ref != "":
			var err error
			ref, err := processAsGoTemplate(flowfileDir, string(e.Ref), templateData)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("unable to process %s executable %d", stage, i))
			}
			exec, err = execUtils.ExecutableForRef(ctx, executable.Ref(ref.String()))
			if err != nil {
				return err
			}
		case e.Cmd != "":
			cmd, err := processAsGoTemplate(flowfileDir, e.Cmd, templateData)
			if err != nil {
				return errors.Wrap(err, fmt.Sprintf("unable to process %s executable %d", stage, i))
			}
			exec = execUtils.ExecutableForCmd(templateParent(ws.AssignedName(), ws.Location(), flowfileDir), cmd.String(), i)
		default:
			return errors.New("post-run executable must have a ref or cmd")
		}
		execEnv := make(map[string]string)
		ee := expressionEnv(templateData)
		maps.Copy(execEnv, ee)
		if len(e.Args) > 0 {
			args := make([]string, 0)
			for _, arg := range e.Args {
				a, err := processAsGoTemplate(flowfileDir, arg, templateData)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("unable to process %s executable %d", stage, i))
				}
				args = append(args, a.String())
			}
			a, err := argUtils.ProcessArgs(exec, args, ee)
			if err != nil {
				ctx.Logger.Error(err, "unable to process arguments")
			}
			maps.Copy(execEnv, a)
		}
		exec.Exec.SetLogFields(map[string]interface{}{
			"stage": stage,
			"step":  i + 1,
		})
		if err := runner.Exec(ctx, exec, engine.NewExecEngine(), execEnv); err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to execute %s executable %d", stage, i))
		}
	}
	return nil
}

func parseSourcePath(
	logger tuikitIO.Logger,
	name, flowFileSrc, wsDir string,
	artifact executable.Artifact,
	templateData expressionData,
) (string, error) {
	var err error
	if artifact.SrcDir != "" {
		flowFileSrc = utils.ExpandDirectory(logger, artifact.SrcDir, wsDir, flowFileSrc, expressionEnv(templateData))
	}
	var sb *bytes.Buffer
	sb, err = processAsGoTemplate(name, filepath.Join(flowFileSrc, artifact.SrcName), templateData)
	if err != nil {
		return "", errors.Wrap(err, "unable to process artifact as template")
	}
	return sb.String(), nil
}

func parseDestinationPath(
	logger tuikitIO.Logger,
	name, dstDir, flowFileSrc, wsDir string,
	artifact executable.Artifact,
	templateData expressionData,
) (string, error) {
	var err error
	if artifact.DstDir != "" {
		dstDir = utils.ExpandDirectory(logger, artifact.DstDir, wsDir, flowFileSrc, expressionEnv(templateData))
	}
	dstName := artifact.DstName
	var db *bytes.Buffer
	db, err = processAsGoTemplate(name, dstName, templateData)
	if err != nil {
		return "", errors.Wrap(err, "unable to process artifact as template")
	}
	dstName = db.String()
	return filepath.Join(dstDir, dstName), nil
}

func templateToFlowfile(
	t *executable.Template,
	templateData expressionData,
) (*executable.FlowFile, error) {
	buf, err := processAsGoTemplate(t.Name(), t.Template, templateData)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("flowfile template %s", t.Name()))
	}

	flowfile := &executable.FlowFile{}
	if err := yaml.NewDecoder(buf).Decode(flowfile); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("unable to decode %s flowfile template", t.Name()))
	}

	return flowfile, nil
}

func processAsGoTemplate(fileName, txt string, data expressionData) (*bytes.Buffer, error) {
	tmpl := expr.NewTemplate(fileName, data)
	if err := tmpl.Parse(txt); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("unable to parse %s template", fileName))
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("unable to execute %s template", fileName))
	}

	return &buf, nil
}

// templateParent returns a pseudo-executable that can be used as a parent for other executables. It simply includes
// the executable context that is derived from the rendered template.
func templateParent(ws, wsPath, flowfilePath string) *executable.Executable {
	e := &executable.Executable{}
	e.SetContext(ws, wsPath, "flow-internal", flowfilePath)
	return e
}
