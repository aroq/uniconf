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
	"github.com/aroq/uniconf/unitool"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Phase struct {
	Name        string
	Args        []interface{}
	Callback    func([]interface{}) (interface{}, error)
	Phases      []*Phase
	ParentPhase *Phase
	Result      *interface{}
	Error       *error
}

type Callback struct {
	Args   []interface{}
	Method func([]interface{}) (interface{}, error)
}

type Uniconf struct {
	config     map[string]interface{}
	sources    map[string]SourceHandler
	flatConfig map[string]interface{}
	//contexts     []string
	phases       map[string]*Phase
	phasesList   []*Phase
	currentPhase *Phase
	rootSource   SourceHandler
}

var u *Uniconf

//var configProviders []func() interface{}

const (
	appTempFilesPath       = ".unipipe_temp"
	sourceMapElementName   = "sources"
	IncludeListElementName = "from"
	sourcesStoragePath     = "sources"
	mainConfigFileName     = "config.yaml"
	includesPath           = "scenarios"
)

// New returns an initialized Uniconf instance.
func New() *Uniconf {
	u = new(Uniconf)
	u.config = make(map[string]interface{})
	u.sources = make(map[string]SourceHandler)
	u.phasesList = make([]*Phase, 0)
	u.phases = make(map[string]*Phase)
	return u
}

// Init initializes uniconf.
func init() {
	log.SetLevel(log.DebugLevel)
	u = New()
}

func phaseFullName(phase *Phase) string {
	name := ""
	if phase.ParentPhase != nil {
		name = phaseFullName(phase.ParentPhase) + "."
	}
	name += phase.Name
	return name
}

func SetRootSource(sourceName string) { u.setRootSource(sourceName) }
func (u *Uniconf) setRootSource(sourceName string) {
	if source := u.getSource(sourceName); source != nil {
		u.rootSource = source
	} else {
		log.Errorf("source %s is not found", sourceName)
	}
}

func Execute() { u.execute(nil, u.phasesList) }
func (u *Uniconf) execute(parentPhase *Phase, phases []*Phase) {
	for _, phase := range phases {
		phase.ParentPhase = parentPhase
		u.currentPhase = phase
		log.Debugf("Execute phase: %s", phaseFullName(phase))
		if phase.Callback != nil {
			result, err := phase.Callback(phase.Args)
			if err != nil {
				log.Errorf("error: %v", err)
			}
			if phase.Result != nil {
				*phase.Result = result
			}
			if phase.Error != nil {
				phase.Error = &err
			}
		}
		if phase.Phases != nil {
			u.execute(phase, phase.Phases)
		}
	}
}

func Config() map[string]interface{} { return u.Config() }
func (u *Uniconf) Config() map[string]interface{} {
	return u.config
}

func (u *Uniconf) mergeConfigEntity(configEntity *ConfigEntity) {
	unitool.Merge(u.config, configEntity.config, true)
}

func AddSource(source SourceHandler) { u.addSource(source) }
func (u *Uniconf) addSource(source SourceHandler) {
	if _, ok := u.sources[source.Name()]; !ok {
		u.sources[source.Name()] = source
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
	}

	log.Fatalf("Source: %s is not registered", name)
	return nil
}

func (u *Uniconf) allSettings(v *viper.Viper) map[string]interface{} {
	result := make(map[string]interface{})
	keys := v.AllKeys()
	for _, value := range keys {
		result[value] = v.Get(value)
	}
	return result
}
