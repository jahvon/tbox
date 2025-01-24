package request

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"

	"github.com/jahvon/flow/internal/context"
	"github.com/jahvon/flow/internal/runner"
	"github.com/jahvon/flow/internal/runner/engine"
	"github.com/jahvon/flow/internal/services/expr"
	"github.com/jahvon/flow/internal/services/rest"
	"github.com/jahvon/flow/types/executable"
)

type requestRunner struct{}

func NewRunner() runner.Runner {
	return &requestRunner{}
}

func (r *requestRunner) Name() string {
	return "request"
}

func (r *requestRunner) IsCompatible(executable *executable.Executable) bool {
	if executable == nil || executable.Request == nil {
		return false
	}
	return true
}

func (r *requestRunner) Exec(
	ctx *context.Context,
	e *executable.Executable,
	_ engine.Engine,
	inputEnv map[string]string,
) error {
	requestSpec := e.Request
	envMap, err := runner.BuildEnvMap(ctx.Logger, e.Env(), inputEnv, runner.DefaultEnv(ctx, e))
	if err != nil {
		return errors.Wrap(err, "unable to set parameters to env")
	}

	url := expandEnvVars(envMap, requestSpec.URL)
	body := expandEnvVars(envMap, requestSpec.Body)
	for key, value := range requestSpec.Headers {
		requestSpec.Headers[key] = expandEnvVars(envMap, value)
	}
	restRequest := rest.Request{
		URL:     url,
		Method:  string(requestSpec.Method),
		Headers: requestSpec.Headers,
		Body:    body,
		Timeout: requestSpec.Timeout,
	}
	resp, err := rest.SendRequest(&restRequest, requestSpec.ValidStatusCodes)
	if err != nil {
		return errors.Wrap(err, "request failed")
	}

	respStr := resp.Body
	if requestSpec.TransformResponse != "" {
		respStr, err = expr.EvaluateString(requestSpec.TransformResponse, resp)
		if err != nil {
			return errors.Wrap(err, "unable to transform response")
		}
	}

	logger := ctx.Logger
	if requestSpec.LogResponse {
		logger.Infox(fmt.Sprintf("Successfully sent request to %s", requestSpec.URL), "response", respStr)
	} else {
		logger.Infof("Successfully sent request to %s", requestSpec.URL)
	}

	if requestSpec.ResponseFile != nil && requestSpec.ResponseFile.Filename != "" {
		targetDir, isTmp, err := requestSpec.ResponseFile.Dir.ExpandDirectory(
			ctx.Logger,
			e.WorkspacePath(),
			e.FlowFilePath(),
			ctx.ProcessTmpDir,
			envMap,
		)
		if err != nil {
			return errors.Wrap(err, "unable to expand directory")
		} else if isTmp {
			ctx.ProcessTmpDir = targetDir
		}

		err = writeResponseToFile(
			respStr,
			filepath.Join(targetDir, requestSpec.ResponseFile.Filename),
			requestSpec.ResponseFile.SaveAs,
		)
		if err != nil {
			return errors.Wrap(err, "unable to save response")
		} else {
			logger.Infof("Successfully saved response to %s", requestSpec.ResponseFile.Filename)
		}
	}

	return nil
}

func writeResponseToFile(resp, responseFile string, format executable.RequestResponseFileSaveAs) error {
	var formattedResp string
	switch format {
	case "", executable.RequestResponseFileSaveAsRaw:
		formattedResp = resp
	case executable.RequestResponseFileSaveAsJson:
		var js map[string]interface{}
		if json.Unmarshal([]byte(resp), &js) != nil {
			return errors.New("response is not a valid JSON string")
		}
		formattedResp = resp
	case executable.RequestResponseFileSaveAsIndentedJson, "formatted-json":
		var respMap map[string]interface{}
		err := json.Unmarshal([]byte(resp), &respMap)
		if err != nil {
			return errors.New("response is not a valid JSON string")
		}
		formattedStr, err := json.MarshalIndent(respMap, "", "  ")
		if err != nil {
			return err
		}
		formattedResp = string(formattedStr)
	case executable.RequestResponseFileSaveAsYaml, executable.RequestResponseFileSaveAsYml:
		var respMap map[string]interface{}
		err := json.Unmarshal([]byte(resp), &respMap)
		if err != nil {
			return errors.New("response is not a valid JSON string")
		}
		yamlStr, err := yaml.Marshal(respMap)
		if err != nil {
			return err
		}
		formattedResp = string(yamlStr)
	default:
		return fmt.Errorf("unsupported output format - %s", format)
	}

	file, err := os.Create(filepath.Clean(responseFile))
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(formattedResp)
	if err != nil {
		return err
	}
	return nil
}

func expandEnvVars(envMap map[string]string, value string) string {
	if envMap == nil || value == "" {
		return value
	}
	return os.Expand(value, func(envVar string) string {
		return envMap[envVar]
	})
}
