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
)

type Uniconf struct {
	config     map[string]interface{}
	configFile string
	sources    map[string]*Source
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
	u.load()
}

// New returns an initialized Uniconf instance.
func New() *Uniconf {
	u := new(Uniconf)
	u.config = make(map[string]interface{})
	u.configFile = path.Join(configFilesPath, mainConfigFileName)

	return u
}

// load loads and processes configuration.
func (u *Uniconf) load() {
	if len(u.config) == 0 {
		u.sources = make(map[string]*Source)

		source := &Source{name: "uniconf", path: "", isLoaded: true}
		u.sources["uniconf"] = source

		// TODO: check if this is needed.
		os.RemoveAll(appTempFilesPath)

		// TODO: refactor to provide settings from outside the app.
		loaders := []map[string]interface{}{
			{"name": configEnvVarName, "configType": "env_var", "source": source, "id": configEnvVarName, "format": "json"},
			{"name": u.configFile, "configType": "file", "source": source, "id": u.configFile, "format": "yaml"},
		}
		for i := 0; i < len(loaders); i++ {
			c1 := loadConfigEntity(loaders[i])
			unitools.Merge(u.config, c1.config)
		}

	}
}

func (u *Uniconf) getSource(name string) *Source {
	if source, ok := u.sources[name]; ok {
		if !source.isLoaded {
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

// SetConfigFile explicitly defines the path, name and extension of the uniconf file.
func SetConfigFile(in string) { u.SetConfigFile(in) }
func (u *Uniconf) SetConfigFile(in string) {
	if in != "" {
		u.configFile = in
	}
}

func GetYaml() (yamlString string) { return u.GetYaml() }
func (u *Uniconf) GetYaml() string {
	y, err := yaml.Marshal(u.config)
	if err != nil {
		log.Fatalf("Err: %v", err)
	}
	return "---\n" + string(y)
}

func GetJson() (yamlString string) { return u.GetJson() }
func (u *Uniconf) GetJson() string {
	y, err := json.Marshal(u.config)
	if err != nil {
		log.Fatalf("Err: %v", err)
	}
	return string(y)
}
