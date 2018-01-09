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
	"fmt"
	"github.com/aroq/uniconf/unitool"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"strings"
)

type Uniconf struct {
	config     map[string]interface{}
	configFile string
	sources    map[string]SourceHandler
	history    map[string]interface{}
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
	//log.SetLevel(log.WarnLevel)
	u = New()
	u.load(u.defaultConfig())
}

// New returns an initialized Uniconf instance.
func New() *Uniconf {
	u := new(Uniconf)
	u.config = make(map[string]interface{})
	u.history = make(map[string]interface{})
	u.configFile = path.Join(configFilesPath, mainConfigFileName)
	u.sources = make(map[string]SourceHandler)
	return u
}

// load loads configuration.
func (u *Uniconf) load(defaultConfig []map[string]interface{}) {
	if len(u.config) == 0 {
		// TODO: check if this is needed.
		os.RemoveAll(appTempFilesPath)

		for i := 0; i < len(defaultConfig); i++ {
			sourceName := defaultConfig[i]["sourceName"].(string)
			sourceType := defaultConfig[i]["sourceType"].(string)

			if sourceType == "env" {
				u.sources[sourceName] = NewSourceEnv("env", nil)
			}
			if sourceType == "file" {
				u.sources[sourceName] = NewSourceFile("project", map[string]interface{}{"path": "."})
			}

			for j := 0; j < len(defaultConfig[i]["configs"].([]map[string]interface{})); j++ {
				if c, err := u.sources[sourceName].LoadConfigEntity(defaultConfig[i]["configs"].([]map[string]interface{})[j]); err == nil {
					unitool.Merge(u.config, c.config)
					unitool.Merge(u.history, c.history)
				}
			}
		}
		if len(u.history) > 0 {
			u.config["history"] = u.history
		}
	}
}
func Config() interface{} { return u.Config() }
func (u *Uniconf) Config() interface{} {
	return u.config
}

func (u *Uniconf) defaultConfig() []map[string]interface{} {
	// TODO: refactor to provide settings from outside the app.

	return []map[string]interface{}{
		{
			"sourceName": "env",
			"sourceType": "env",
			"configs": []map[string]interface{}{
				{
					"id": configEnvVarName,
				},
			},
		},
		{
			"sourceName": "project",
			"sourceType": "file",
			"configs": []map[string]interface{}{
				{
					"id": u.configFile,
				},
			},
		},
	}
}

func (u *Uniconf) getSource(name string) SourceHandler {
	if source, ok := u.sources[name]; ok {
		if !source.IsLoaded() {
			// Lazy load source.
			err := source.LoadSource()
			if err != nil {
				log.Fatalf("Source: %s was not loaded because of source.getSource() error: %v", name, err)
			}
		}
		return source
	} else {
		log.Fatalf("Source: %s is not registered", name)
		return nil
	}
}

func Collect(jsonPath, key string) string { return u.collect(jsonPath, key) }
func (u *Uniconf) collect(jsonPath, key string) string {
	result, _ := unitool.CollectKeyParamsFromJsonPath(u.config, jsonPath, key)
	return unitool.MarshallYaml(result)
}

func Explain(jsonPath, key string) { u.explain(jsonPath, key) }
func (u *Uniconf) explain(jsonPath, key string) {
	result := unitool.SearchMapWithPathStringPrefixes(u.config, jsonPath)
	fmt.Println("Result:")
	fmt.Println(unitool.MarshallYaml(result))

	if history, ok := u.config["history"].(map[string]interface{})[strings.Trim(jsonPath, ".")]; ok {
		fmt.Println("Load history:")
		fmt.Println(unitool.MarshallYaml(history))
	}

	u.process(result, "", "config")
	fmt.Println("From processed result:")
	fmt.Println(unitool.MarshallYaml(result))
}

// SetConfigFile explicitly defines the path, name and extension of the uniconf file.
func SetConfigFile(in string) { u.SetConfigFile(in) }
func (u *Uniconf) SetConfigFile(in string) {
	if in != "" {
		u.configFile = in
	}
}

func GetYaml() (yamlString string) { return u.getYaml() }
func (u *Uniconf) getYaml() string {
	return unitool.MarshallYaml(u.config)
}

func GetJson() (yamlString string) { return u.getJson() }
func (u *Uniconf) getJson() string {
	return unitool.MarshallJson(u.config)
}

