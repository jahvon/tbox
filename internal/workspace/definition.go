package workspace

import (
	stdErrors "errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/jahvon/flow/internal/errors"
	"github.com/jahvon/flow/internal/executable"
	"github.com/jahvon/flow/internal/executable/consts"
	"github.com/jahvon/flow/internal/io"
)

const (
	definitionExt = ".flow"
)

var (
	log = io.Log().With().Str("pkg", "workspace").Logger()
	up  = ".."
)

type Definition struct {
	Namespace   string          `yaml:"namespace"`
	Tags        []string        `yaml:"tags"`
	Executables executable.List `yaml:"executables"`

	workspace     string
	workspacePath string
}

func (d *Definition) SetContext(workspace, workspacePath string) {
	d.workspace = workspace
	d.workspacePath = workspacePath
	for _, exec := range d.Executables {
		exec.SetContext(workspace, workspacePath, d.Namespace)
	}
}

func (d *Definition) HasAnyTag(tags []string) bool {
	if len(tags) == 0 {
		return true
	}

	for _, t := range tags {
		if d.HasTag(t) {
			return true
		}
	}
	return false
}

func (d *Definition) HasTag(tag string) bool {
	for _, t := range d.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

type DefinitionList []*Definition

func (l *DefinitionList) FilterByNamespace(namespace string) DefinitionList {
	if namespace == "" {
		return *l
	}

	var definitions []*Definition
	for _, definition := range *l {
		if definition.Namespace == namespace {
			definitions = append(definitions, definition)
		}
	}

	log.Trace().
		Int("definitions", len(definitions)).
		Msg("filtered definitions by namespace")
	return definitions
}

func (l *DefinitionList) FilterByTag(tag string) DefinitionList {
	if tag == "" {
		return *l
	}

	var definitions []*Definition
	for _, definition := range *l {
		for _, definitionTag := range definition.Tags {
			if definitionTag == tag {
				definitions = append(definitions, definition)
			}
		}
	}

	log.Trace().
		Int("definitions", len(definitions)).
		Msg("filtered definitions by tag")
	return definitions
}

// LookupExecutableByTypeAndName searches for an executable by type and name.
// If the executable is found, the namespace and executable are returned.
func (l *DefinitionList) LookupExecutableByTypeAndName(
	agent consts.AgentType,
	name string,
) (string, *executable.Executable, error) {
	log.Debug().Msgf("looking up executable %s of type %s", name, agent)

	notFound := errors.ExecutableNotFoundError{Agent: agent, Name: name}
	for _, definition := range *l {
		exec, err := definition.Executables.FindByTypeAndName(agent, name)
		if err != nil && !stdErrors.Is(err, notFound) {
			log.Err(err).Msgf("failed to lookup executable %s of type %s", name, agent)
			continue
		} else if exec != nil {
			return definition.Namespace, exec, nil
		}
	}
	return "", nil, notFound
}

func LoadDefinitions(workspace, workspacePath string) (DefinitionList, error) {
	definitionFiles, err := findDefinitionFiles(workspace, workspacePath)
	if err != nil {
		return nil, err
	}

	var definitions []*Definition
	for _, definitionFile := range definitionFiles {
		definition, err := loadDefinition(definitionFile)
		if err != nil {
			return nil, err
		}
		definition.SetContext(workspace, workspacePath)
		definitions = append(definitions, definition)
	}
	log.Trace().Msgf("loaded %d definitions", len(definitions))

	return definitions, nil
}

func loadDefinition(definitionFile string) (*Definition, error) {
	file, err := os.Open(filepath.Clean(definitionFile))
	if err != nil {
		return nil, fmt.Errorf("unable to open definition file - %w", err)
	}
	defer file.Close()

	config := &Definition{}
	err = yaml.NewDecoder(file).Decode(config)
	if err != nil {
		return nil, fmt.Errorf("unable to decode definition file - %w", err)
	}

	return config, nil
}

func findDefinitionFiles(workspace, workspacePath string) ([]string, error) { //nolint:gocognit
	wsCfg, err := LoadConfig(workspace, workspacePath)
	if err != nil {
		return nil, err
	}

	var includePaths, excludedPaths []string
	if wsCfg.Executables != nil {
		includePaths = wsCfg.Executables.Included
		if len(includePaths) == 0 {
			includePaths = []string{workspacePath}
		} else {
			for i, path := range includePaths {
				includePaths[i] = filepath.Clean(path)
			}
		}

		excludedPaths = wsCfg.Executables.Excluded
		if len(excludedPaths) > 0 {
			for i, path := range excludedPaths {
				excludedPaths[i] = filepath.Clean(path)
			}
		}
	} else {
		includePaths = []string{workspacePath}
	}

	var definitions []string
	walkDirFunc := func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if excludedPathMatches(path, excludedPaths) {
			return filepath.SkipDir
		}
		if !includePathMatches(path, includePaths) {
			return nil
		}
		if filepath.Ext(entry.Name()) == definitionExt {
			log.Trace().Msgf("found definition file %s", path)
			definitions = append(definitions, path)
		}
		return nil
	}

	if err := filepath.WalkDir(workspacePath, walkDirFunc); err != nil {
		return nil, err
	}
	return definitions, nil
}

func includePathMatches(path string, includePaths []string) bool {
	if includePaths == nil {
		return true
	}

	for _, includePath := range includePaths {
		rel, err := filepath.Rel(includePath, path)
		if err != nil {
			log.Err(err).Msgf("unable to get relative path for %s", path)
			continue
		}

		if path == includePath || !strings.HasPrefix(rel, up) {
			return true
		}
	}
	return false
}

func excludedPathMatches(path string, excludedPaths []string) bool {
	if excludedPaths == nil {
		return false
	}

	for _, excludedPath := range excludedPaths {
		rel, err := filepath.Rel(excludedPath, path)
		if err != nil {
			log.Err(err).Msgf("unable to get relative path for %s", path)
			continue
		}

		if rel != up && (path == excludedPath || strings.HasPrefix(rel, up)) {
			return true
		}
	}
	return false
}
