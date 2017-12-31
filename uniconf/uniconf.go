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
	"log"
	"os"
	"path"
	"strings"
)

type Uniconf struct {
	config            map[string]interface{}
	configFile        string
	registeredSources map[string]Source
	loadedSources     map[string]Source
}

var u *Uniconf

const (
	appTempFilesPath       = ".unipipe_temp"
	configFilesPath        = ".unipipe"
	sourceMapElementName   = "sources"
	IncludeListElementName = "from"
	sourcesStoragePath     = "sources"
	mainConfigFileName     = "config.yaml"
	includesPath           = "scenarios"
	configEnvVarName       = "UNIPIPE_CONFIG"
)

const (
	refPrefix     = "refs/"
	refHeadPrefix = refPrefix + "heads/"
	//refTagPrefix    = refPrefix + "tags/"
)

func init() {
	u = New()
}

// New returns an initialized Uniconf instance.
func New() *Uniconf {
	u := new(Uniconf)
	u.config = make(map[string]interface{})
	u.configFile = path.Join(configFilesPath, mainConfigFileName)

	return u
}

func (u *Uniconf) RegisterSource(name string, sourceMap map[string]interface{}) {
	source := NewSource(name, sourceMap)
	u.registeredSources[name] = *source
}

func (u *Uniconf) GetSource(name string) *Source {
	if source, ok := u.registeredSources[name]; ok {
		return &source
	} else {
		return nil
	}
}

func (u *Uniconf) RegisterSources(sources map[string]interface{}) {
	for k, v := range sources {
		log.Printf("Processing source: %s\n", k)
		if _, ok := u.registeredSources[k]; !ok {
			log.Printf("Source is not loaded: %s\n", k)
			u.RegisterSource(k, v.(map[string]interface{}))
		} else {
			log.Printf("Source: %s already loaded", k)
		}
	}
}

func (u *Uniconf) LoadSource(name string) *Source {
	source := u.GetSource(name)
	if source != nil {
		if _, ok := u.loadedSources[name]; !ok {
			err := source.LoadSource()
			if err != nil {
				log.Fatalf("Source: %s was not loaded because of source.LoadSource() error: %v", name, err)
			} else {
				u.loadedSources[name] = *source
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
		u.RegisterSources(sources)
	}
}

func (u *Uniconf) ProcessIncludes(config map[string]interface{}, currentSourceName string) (map[string]interface{}) {
	includesConfig := make(map[string]interface{})

	if includes, ok := config[IncludeListElementName]; ok {
		for _, include := range includes.([]interface{}) {
			scenario := include.(string)
			log.Printf("Processing include: %s", scenario)
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

			source := u.LoadSource(sourceName)

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

func (u *Uniconf) ProcessConfig(config map[string]interface{}, currentSourceName string) (map[string]interface{}) {
	u.ProcessSources(config)
	includesConfig := u.ProcessIncludes(config, currentSourceName)
	return unitools.Merge(includesConfig, config)
}

func (u *Uniconf) LoadConfig(yamlFile []byte, currentSourceName string) map[string]interface{} {
	config, _ := unitools.UnmarshalYaml(yamlFile)
	processedConfig := u.ProcessConfig(config, currentSourceName)
	return processedConfig
}

func (u *Uniconf) Load() {
	if len(u.config) == 0 {
		u.registeredSources = make(map[string]Source)
		u.registeredSources["."] = Source{Path: "."}
		u.loadedSources = make(map[string]Source)
		u.loadedSources["."] = Source{Path: "."}

		// TODO: check if this is needed.
		os.RemoveAll(appTempFilesPath)

		log.Printf("Processing file: %v", u.configFile)
		yamlFile := unitools.ReadFile(u.configFile)

		if envConfig, err := unitools.UnmarshalEnvVarJson(configEnvVarName); err == nil {
			u.loadedSources["env"] = Source{Path: "."}
			processedConfig := u.ProcessConfig(envConfig, "env")
			unitools.Merge(u.config, processedConfig)
		}

		projectConfig := u.LoadConfig(yamlFile, ".")
		unitools.Merge(projectConfig, u.config)
		u.config = projectConfig
	}
}

// SetConfigFile explicitly defines the path, name and extension of the uniconf file.
func SetConfigFile(in string) { u.SetConfigFile(in) }
func (u *Uniconf) SetConfigFile(in string) {
	if in != "" {
		u.configFile = in
	}
}

func Yaml() (yamlString string) { return u.Yaml() }
func (u *Uniconf) Yaml() (yamlString string) {
	u.Load()
	y, err := yaml.Marshal(u.config)
	if err != nil {
		log.Fatalf("Err: %v\n", err)

	}
	yamlString = string(y)
	return "---\n" + yamlString
}

func (u *Uniconf) Json() (jsonString string) {
	u.Load()
	y, err := json.Marshal(u)
	if err != nil {
		log.Fatalf("Err: %v\n", err)

	}
	jsonString = string(y)
	return jsonString
}
