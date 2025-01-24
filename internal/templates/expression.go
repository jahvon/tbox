package templates

import "runtime"

type expressionData map[string]interface{}

func newExpressionData(
	ws, wsPath, flowfileName, flowfileDir, flowfilePath, templatePath string,
	envMap, formMap map[string]string,
) expressionData {
	return map[string]interface{}{
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"workspace":     ws,
		"workspacePath": wsPath,
		"name":          flowfileName,
		"directory":     flowfileDir,
		"flowFilePath":  flowfilePath,
		"templatePath":  templatePath,
		"env":           envMap,
		"form":          formMap,
	}
}

func expressionEnv(data expressionData) map[string]string {
	if env, ok := data["env"].(map[string]string); ok {
		return env
	}
	return nil
}
