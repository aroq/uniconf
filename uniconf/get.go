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
	"log"
	"path"
	"os"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"strings"
	"encoding/json"
	"github.com/ghodss/yaml"
	"io/ioutil"
	"fmt"
)

//type ProjectConf map[string]interface{}
type Conf map[string]interface{}

var c Conf

type Source struct {
	Url          string
	Ref string
	StoredPath   string
	ScenariosDir string
}

var loadedSources map[string]Source

const (
	appTempFilesPath       = ".unipipe_temp"
	configFilesPath        = ".unipipe"
	sourceListElementName  = "sources"
	sourcesListElementName = "from"
	sourcesStoragePath     = "sources"
	mainConfigFileName     = "config.yaml"
	scenariosPath          = "scenarios"
	configEnvVarName       = "UNIPIPE_SOURCES"
)

// TODO: load default config values from yamlin.yaml.
type Config struct {
	sources_list_key_name  string
	sources_storage_path   string
	scenarios_key_name     string
	scenarios_default_path string
	main_config_file_name  string
}

func (c *Conf) ReadFile(filename string) []byte {
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Panicf("yamlFile.Get err: %v ", err)
	}
	return yamlFile
}

func (c *Conf) Unmarshal(yamlFile []byte) *Conf {
	err := yaml.Unmarshal(yamlFile, c)
	if err != nil {
		log.Panicf("Unmarshal: %v", err)
	}
	return c
}

func (c *Conf) Process(yamlFile []byte, currentSourceName string) Conf {
	config := make(map[string]interface{})

	envConfigString := os.Getenv("UNIPIPE_CONFIG")
	if envConfigString != "" {
		var envConfig Conf
		json.Unmarshal([]byte(envConfigString), &envConfig)
		Merge(config, envConfig)
	}

	var projectConfig Conf
	projectConfig.Unmarshal(yamlFile)

	Merge(config, projectConfig)

	if sources, ok := config[sourceListElementName].(map[string]interface{}); ok {
		for k, v := range sources {
			log.Printf("Processing source: %s\n", k)
			sourcePath := path.Join(appTempFilesPath, sourcesStoragePath, k)
			if _, ok := loadedSources[k]; !ok {
				log.Printf("Source is not loaded: %s\n", k)
				source := v.(map[string]interface{})
				repo := source["repo"].(string)
				prefix, ok := source["prefix"]
				if !ok {
					prefix = "refs/heads/"
				}
				var reference string
				if _, ok := source["ref"]; ok {
					reference = source["ref"].(string)
				} else {
					reference = "master"
				}
				log.Printf("Cloning source repo: %s, reference: %s\n", repo, reference)
				log.Println("Try to clone without 'git' command...")

				_, err := git.PlainClone(sourcePath, false, &git.CloneOptions{
					URL:          repo,
					Progress:     os.Stdout,
					SingleBranch: true,
					Depth:        1,
					ReferenceName: plumbing.ReferenceName(prefix.(string) + reference),
				})
				if err != nil {
					log.Printf("Error: %s\n", err)
					// TODO: Recheck this part.
					log.Println("Try to clone with 'git' command execution...")
					ExecCommandString(fmt.Sprintf("git clone --depth=1 -b %s %s %s", reference, repo, sourcePath))
				} else {
					loadedSources[k] = Source{StoredPath: sourcePath, Url: repo, Ref: reference}
				}
			} else {
				log.Printf("Source: %s already loaded", k)
			}
		}
	}

	scenariosConfig := make(Conf)
	if scenarios, ok := projectConfig[sourcesListElementName]; ok {
		for _, scenarioItem := range scenarios.([]interface{}) {
			scenario := scenarioItem.(string)
			log.Printf("Processing scenario: %s", scenario)
			sourceName, scenarioName := "", ""
			if strings.Contains(scenario, ":") {
				s := strings.Split(scenario, ":")
				sourceName, scenarioName = s[0], s[1]
			} else {
				sourceName, scenarioName = currentSourceName, scenario
			}
			if !(strings.Index(scenarioName, "/") == 0) {
				scenarioName = path.Join(scenariosPath, scenarioName)
			}
			sourcePath := loadedSources[sourceName].StoredPath

			var scenarioFileNamesToCheck []string

			scenarioFileName := path.Join(sourcePath, scenarioName)
			scenarioFileNamesToCheck = append(scenarioFileNamesToCheck, scenarioFileName+".yaml")
			scenarioFileNamesToCheck = append(scenarioFileNamesToCheck, path.Join(scenarioFileName, mainConfigFileName))

			for _, f := range scenarioFileNamesToCheck {
				if _, err := os.Stat(f); err == nil {
					yamlSubFile := c.ReadFile(f)
					scenarioConfig := c.Process(yamlSubFile, sourceName)
					Merge(scenariosConfig, scenarioConfig)
				}
			}
		}
	}

	Merge(scenariosConfig, projectConfig)
	return scenariosConfig
}

func (c *Conf) Get() {
	if len(*c) == 0 {
		loadedSources = make(map[string]Source)
		loadedSources["."] = Source{StoredPath: "."}

		os.RemoveAll(appTempFilesPath)

		filename := path.Join(configFilesPath, mainConfigFileName)

		log.Printf("Processing file: %v", filename)
		yamlFile := c.ReadFile(filename)

		*c = c.Process(yamlFile, ".")
	}
}

func Yaml() (yamlString string) { return c.Yaml() }
func (c *Conf) Yaml() (yamlString string) {
	c.Get()
	y, err := yaml.Marshal(c)
	if err != nil {
		log.Fatalf("Err: %v\n", err)

	}
	yamlString = string(y)
	return "---\n" + yamlString
}

func (c *Conf) Json() (jsonString string) {
	c.Get()
	y, err := json.Marshal(c)
	if err != nil {
		log.Fatalf("Err: %v\n", err)

	}
	jsonString = string(y)
	return jsonString
}
