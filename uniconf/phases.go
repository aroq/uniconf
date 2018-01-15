package uniconf

import (
	"os"
	"github.com/aroq/uniconf/unitool"
	"github.com/spf13/viper"
	"bytes"
	log "github.com/sirupsen/logrus"
	"strings"
	"strconv"
	"fmt"
)

func (u *Uniconf) setCurrentPhase(name string) {
	u.currentPhase = u.phases[name]
}

func AddPhase(phase *Phase) { u.addPhase(phase) }
func (u *Uniconf) addPhase(phase *Phase) {
	u.phases[phase.Name] = phase
	u.phasesList = append(u.phasesList, phase)
}

// Load loads configuration.
func Load(inputs []interface{}) (interface{}, error) { return u.load(inputs) }
func (u *Uniconf) load(inputs []interface{})(interface{}, error) {
	defaultConfig := make(map[string]interface{})
	for _, p := range configProviders {
		c := p().(map[string]interface{})
		unitool.Merge(defaultConfig, c)
	}
	if len(u.config) == 0 {
		os.RemoveAll(appTempFilesPath)
		source, err := AddSource(NewSource("root", nil))
		if err != nil {
			log.Errorf("Root source was not added")
		}
		configMap := map[string]interface{}{
			"id": "root",
			"config": defaultConfig,
		}
		if c, err := source.LoadConfigEntity(configMap); err == nil {
			u.mergeConfigEntity(c)
		} else {
			log.Errorf("Config entity is not loaded")
		}
	}
	return nil, nil
}

func PrintHistory(inputs []interface{}) (interface{}, error) { return u.printHistory(inputs) }
func (u *Uniconf) printHistory(inputs []interface{})(interface{}, error) {
	if len(u.history) > 0 {
		fmt.Println("Config history:")
		fmt.Println(unitool.MarshallYaml(u.history))
	}
	return nil, nil
}

func ProcessContexts(inputs []interface{}) (interface{}, error) { return u.processContexts(inputs) }
func (u *Uniconf) processContexts(inputs []interface{})(interface{}, error) {
	for _, c := range u.contexts {
		context, _ := unitool.CollectInvertedKeyParamsFromJsonPath(u.config, c, "context")
		if context != nil {
			unitool.Merge(u.config, context)
		}
	}
	return nil, nil
}

func FlattenConfig(inputs []interface{}) (interface{}, error) { return u.flattenConfig(inputs) }
func (u *Uniconf) flattenConfig(inputs []interface{})(interface{}, error) {
	viper := viper.New()
	var yamlConfig =[]byte(GetYaml())
	viper.SetConfigType("yaml")
	viper.ReadConfig(bytes.NewBuffer(yamlConfig))
	u.flatConfig = u.allSettings(viper)
	return nil, nil
}

func PrintConfig(inputs []interface{}) (interface{}, error) { return u.printConfig(inputs) }
func (u *Uniconf) printConfig(inputs []interface{})(interface{}, error) {
	if len(inputs) > 0 {
		path := inputs[0].(string)
		fmt.Println(unitool.MarshallYaml(unitool.SearchMapWithPathStringPrefixes(u.config, path)))
	} else {
	    fmt.Println(unitool.MarshallYaml(u.config))
	}
	return nil, nil
}

func processKeys(key string, source interface{}, parent interface{}, path string, phase *Phase, processors []*Processor, depth int, excludeKeys []string) {
	//log.Debugf("processKeys path: %s, key: %s", path, key)
	if depth > -100 {
		switch source.(type) {
		case string:
			for _, processor := range processors {
				if ((processor.IncludeKeys != nil && stringListContains(processor.IncludeKeys, key)) || processor.IncludeKeys == nil) &&
			    	((processor.ExcludeKeys != nil && !stringListContains(processor.ExcludeKeys, key)) || processor.ExcludeKeys == nil) {
					value := source.(string)
					log.Debugf("Key processing start: %s - %v", path, value)
					result, processed, mergeToParent, removeParentKey, replaceSource := processor.Callback(value, path, phase)
					if mergeToParent {
						unitool.Merge(parent, result)
					}
					if removeParentKey {
						delete(parent.(map[string]interface{}), key)
					}
					if replaceSource != nil {
						value = replaceSource.(string)
					}
					if processed && removeParentKey {
						log.Debugf("Key processed: %s %v", path, value)
						if _, ok := parent.(map[string]interface{})[key + "_processed"]; !ok {
							parent.(map[string]interface{})[key + "_processed"] = make([]string, 0)
							keyProcessed := source.(string) + " (" + value + ")"
							if source.(string) == value {
								keyProcessed = value
							}
							parent.(map[string]interface{})[key + "_processed"] = append(parent.(map[string]interface{})[key + "_processed"].([]string), keyProcessed)
						}
					}
					if replaceSource != nil {
						source = replaceSource
					}
					if mergeToParent {
						parts := strings.Split(path, ".")
						p := strings.Join(parts[:len(parts) - 1], ".")
						processKeys("", parent, source, p, phase, processors, depth, excludeKeys)
					}
				}
			}
		case []interface{}:
			l := source.([]interface{})
			for i := 0; i < len(l); i++ {
				p := path
				switch l[i].(type) {
				case string:
				default:
					p = strings.Join([]string{path, strconv.Itoa(i)}, ".")
				}
				processKeys(key, l[i], parent, p, phase, processors, depth, excludeKeys)
			}
		case map[string]interface{}:
			for k, v := range source.(map[string]interface{}) {
				depth -= 1
				if !stringListContains(excludeKeys, k) {
					processKeys(k, v, source, strings.Join([]string{path, k}, "."), phase, processors, depth, excludeKeys)
				} else {
					log.Debugf("Key skipped as ecluded by parent: %s", k)
				}
			}
		}
	}
}

func stringListContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// Process processes configuration.
func ProcessKeys(inputs []interface{}) (interface{}, error) { return u.processKeys(inputs) }
func (u *Uniconf) processKeys(inputs []interface{})(interface{}, error) {
	path := inputs[0].(string)
	keys := strings.Split(path, ".")
	keyPrefix := inputs[1].(string)
	processors := inputs[2].([]*Processor)
	p := ""
	for _, v := range keys {
		p = strings.Trim(p + ".jobs." + v, ".")
		source := unitool.SearchMapWithPathStringPrefixes(u.config, p)
		processKeys("", source, nil, p, u.currentPhase, processors, 1, []string{keyPrefix})
	}
	return u.config, nil
}
