// Copyright © 2017 Alexander Tolstikov <tolstikov@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package uniconf

import (
	"encoding/json"
	"github.com/aroq/uniconf/unitools"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"strings"
)

type Uniconf struct {
	config        map[string]interface{}
	configFile    string
	sources       map[string]*Source
}

var u *Uniconf

const (
	appTempFilesPath       = ".unipipe_temp"
	configFilesPath        = ".unipipe"
	sourceMapElementName   = "sources"
	includeListElementName = "from"
	sourcesStoragePath     = "sources"
	mainConfigFileName     = "config.yaml"
	includesPath           = "scenarios"
	configEnvVarName       = "UNIPIPE_CONFIG"
)

// Init initializes uniconf.
func init() {
	u = New()
	u.Load()
}

// New returns an initialized Uniconf instance.
func New() *Uniconf {
	u := new(Uniconf)
	u.config = make(map[string]interface{})
	u.configFile = path.Join(configFilesPath, mainConfigFileName)

	return u
}

// Load loads and processes configuration.
func (u *Uniconf) Load() {
	if len(u.config) == 0 {
		u.sources = make(map[string]*Source)
		u.sources["."] = &Source{Path: ".", isLoaded: true}

		// TODO: check if this is needed.
		os.RemoveAll(appTempFilesPath)

		log.Printf("Process file: %v", u.configFile)
		yamlFile := unitools.ReadFile(u.configFile)

		if envConfig, err := unitools.UnmarshalEnvVarJson(configEnvVarName); err == nil {
			u.sources["env"] = &Source{Path: ".", isLoaded: true}
			processedEnvConfig := u.ProcessConfig(envConfig, "env")
			unitools.Merge(u.config, processedEnvConfig)
		}

		projectConfig := u.LoadConfig(yamlFile, ".")
		unitools.Merge(projectConfig, u.config)
		u.config = projectConfig
	}
}

func (u *Uniconf) GetSource(name string) *Source {
	if source, ok := u.sources[name]; ok {
		if !source.isLoaded {
			err := source.LoadSource()
			if err != nil {
				log.Fatalf("Source: %s was not loaded because of source.GetSource() error: %v", name, err)
			}
		}
		return source
	} else {
		log.Fatalf("Source: %s is not registered", name)
		return nil
	}
}

func (u *Uniconf) ProcessSources(config map[string]interface{}) {
	if sources, ok := config[sourceMapElementName].(map[string]interface{}); ok {
		for k, v := range sources {
			log.Printf("Process source: %s", k)
			if _, ok := u.sources[k]; !ok {
				source := NewSource(k, v.(map[string]interface{}))
				u.sources[k] = source
			} else {
				log.Printf("Source: %s already loaded", k)
			}
		}
	}
}

func (u *Uniconf) ProcessIncludes(config map[string]interface{}, currentSourceName string) map[string]interface{} {
	includesConfig := make(map[string]interface{})

	if includes, ok := config[includeListElementName]; ok {
		for _, include := range includes.([]interface{}) {
			scenario := include.(string)
			log.Printf("Process include: %s", scenario)
			sourceName, include := "", ""
			if strings.Contains(scenario, ":") {
				s := strings.Split(scenario, ":")
				sourceName, include = s[0], s[1]
			} else {
				sourceName, include = currentSourceName, scenario
			}
			if !(strings.Index(include, "/") == 0) {
				include = path.Join(includesPath, include)
			}

			source := u.GetSource(sourceName)

			var includeFileNamesToCheck []string
			includeFileName := path.Join(source.Path, include)

			includeFileNamesToCheck = append(includeFileNamesToCheck, includeFileName+".yaml")
			includeFileNamesToCheck = append(includeFileNamesToCheck, includeFileName+".yml")
			includeFileNamesToCheck = append(includeFileNamesToCheck, path.Join(includeFileName, mainConfigFileName))

			for _, f := range includeFileNamesToCheck {
				if _, err := os.Stat(f); err == nil {
					includeConfig := u.LoadConfig(unitools.ReadFile(f), sourceName)
					unitools.Merge(includesConfig, includeConfig)
				}
			}
		}
	}
	return includesConfig
}

func (u *Uniconf) ProcessConfig(config map[string]interface{}, currentSourceName string) map[string]interface{} {
	u.ProcessSources(config)
	includesConfig := u.ProcessIncludes(config, currentSourceName)
	return unitools.Merge(includesConfig, config)
}

func (u *Uniconf) LoadConfig(yamlFile []byte, currentSourceName string) map[string]interface{} {
	config, _ := unitools.UnmarshalYaml(yamlFile)
	processedConfig := u.ProcessConfig(config, currentSourceName)
	return processedConfig
}

// SetConfigFile explicitly defines the path, name and extension of the uniconf file.
func SetConfigFile(in string) { u.SetConfigFile(in) }
func (u *Uniconf) SetConfigFile(in string) {
	if in != "" {
		u.configFile = in
	}
}

func GetYaml() (yamlString string) { return u.GetYaml() }
func (u *Uniconf) GetYaml() (yamlString string) {
	y, err := yaml.Marshal(u.config)
	if err != nil {
		log.Fatalf("Err: %v", err)

	}
	return "---\n" + string(y)
}

func GetJson() (yamlString string) { return u.GetJson() }
func (u *Uniconf) GetJson() (jsonString string) {
	y, err := json.Marshal(u)
	if err != nil {
		log.Fatalf("Err: %v", err)

	}
	return string(y)
}
