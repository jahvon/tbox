package workspace

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	ConfigFileName = "workspace.yaml"
)

type Config struct {
	RegisteredExecutables map[string]string `yaml:"executables"`
}

func LoadWorkspaceConfig(workspacePath string) (*Config, error) {
	file, err := os.Open(workspacePath + "/" + ConfigFileName)
	if err != nil {
		return nil, fmt.Errorf("unable to open workspace config file - %v", err)
	}
	defer file.Close()

	config := &Config{}
	err = yaml.NewDecoder(file).Decode(config)
	if err != nil {
		return nil, fmt.Errorf("unable to decode workspace config file - %v", err)
	}

	return config, nil
}

func WriteWorkspaceConfig(workspacePath string, config *Config) error {
	file, err := os.Create(workspacePath + "/" + ConfigFileName)
	if err != nil {
		return fmt.Errorf("unable to create workspace config file - %v", err)
	}
	defer file.Close()

	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("unable to truncate config file - %v", err)
	}

	err = yaml.NewEncoder(file).Encode(config)
	if err != nil {
		return fmt.Errorf("unable to encode workspace config file - %v", err)
	}

	return nil
}

func CreateWorkspaceDirectory(location string) error {
	if info, err := os.Stat(location); os.IsNotExist(err) {
		err = os.MkdirAll(location, 0755)
		if err != nil {
			return fmt.Errorf("unable to create workspace directory - %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("unable to check for workspace directory - %v", err)
	} else if !info.IsDir() {
		return fmt.Errorf("workspace path (%s) exists but is not a directory", location)
	}

	if configInfo, err := os.Stat(location + "/" + ConfigFileName); os.IsNotExist(err) {
		config := defaultConfig()
		err = WriteWorkspaceConfig(location, config)
		if err != nil {
			return fmt.Errorf("unable to write workspace config file - %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("unable to check for workspace config file - %v", err)
	} else if configInfo.IsDir() {
		return fmt.Errorf("workspace config file (%s) exists but is a directory", location+"/"+ConfigFileName)
	}

	return nil
}

func defaultConfig() *Config {
	return &Config{
		RegisteredExecutables: make(map[string]string),
	}
}
