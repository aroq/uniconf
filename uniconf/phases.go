package uniconf

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/aroq/uniconf/unitool"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

//func (u *Uniconf) setCurrentPhase(name string) {
//	u.currentPhase = u.phases[name]
//}

func AddPhase(phase *Phase) { u.addPhase(phase) }
func (u *Uniconf) addPhase(phase *Phase) {
	u.phases[phase.Name] = phase
	u.phasesList = append(u.phasesList, phase)
}

// Load loads configuration.
func Load(inputs []interface{}) (interface{}, error) { return u.load(inputs) }
func (u *Uniconf) load(inputs []interface{}) (interface{}, error) {
	log.Info("load")
	if len(u.config) == 0 {
		log.Info("config is not loaded yet")
		os.RemoveAll(appTempFilesPath)

		if u.rootSource != nil {
			configMap := map[string]interface{}{
				"id": "root",
			}
			if c, err := u.rootSource.LoadConfigEntity(configMap); err == nil {
				u.mergeConfigEntity(c)
			} else {
				log.Errorf("Config entity is not loaded")
			}
		}
	}
	log.Info("load end")
	return nil, nil
}

func PrintHistory(inputs []interface{}) (interface{}, error) { return u.printHistory(inputs) }
func (u *Uniconf) printHistory(inputs []interface{}) (interface{}, error) {
	if len(u.history) > 0 {
		fmt.Println("Config history:")
		fmt.Println(unitool.MarshallYaml(u.history))
	}
	return nil, nil
}

func DeepCollectChildren(inputs []interface{}) (interface{}, error) {
	return u.deepCollectChildren(inputs)
}
func (u *Uniconf) deepCollectChildren(inputs []interface{}) (interface{}, error) {
	if len(inputs) > 1 {
		path := inputs[0].(string)
		key := inputs[1].(string)
		object, _ := unitool.DeepCollectChildren(u.config, path, key)
		return object, nil
	}
	return nil, nil
}

func ProcessContext(inputs []interface{}) (interface{}, error) { return u.processContext(inputs) }
func (u *Uniconf) processContext(inputs []interface{}) (interface{}, error) {
	if len(inputs) > 1 {
		entityName := inputs[0].(string)
		entityID := inputs[1].(string)
		if _, ok := u.config["entities"]; ok {
			// Get entity handler from the config.
			entityHandler := u.config["entities"].(map[string]interface{})[entityName].(map[string]interface{})
			// childrenKey determines key in config used to hold child items.
			childrenKey := entityHandler["children_key"].(string)

			processors := make([]*Processor, 0)
			if processorsList, ok := entityHandler["processors"]; ok {
				for _, processor := range processorsList.([]interface{}) {
					if processor.(string) == "from_processor" {
						processors = append(
							processors,
							&Processor{
								Callback:    FromProcess,
								IncludeKeys: []string{"from"},
							})
					}
				}
			}
			ProcessKeys([]interface{}{childrenKey, "", processors})

			switch entityHandler["retrieve_handler"].(string) {
			case "DeepCollectChildren":
				entity, _ := unitool.DeepCollectChildren(u.config, entityID, childrenKey)
				u.setContextObject(entityHandler["context_name"].(string), entity)
				return entity, nil
			}
		} else {
			return nil, errors.New("config contexts are not defined")
		}
	}
	return nil, nil
}

func SetContext(inputs []interface{}) (interface{}, error) { return u.setContext(inputs) }
func (u *Uniconf) setContext(inputs []interface{}) (interface{}, error) {
	if len(inputs) > 1 {
		contextName := inputs[0].(string)
		i2 := inputs[1].(*interface{})
		i3 := *i2
		object := i3.(map[string]interface{})
		if object != nil {
			u.setContextObject(contextName, object)
		}
	}
	return nil, nil
}

func (u *Uniconf) setContextObject(contextName string, context map[string]interface{}) {
	if _, ok := u.config["contexts"]; !ok {
		u.config["contexts"] = make(map[string]interface{})
	}
	u.config["contexts"].(map[string]interface{})[contextName] = context
	if context, ok := context["context"]; ok {
		unitool.Merge(u.config, context)
	}
}

func FlattenConfig(inputs []interface{}) (interface{}, error) { return u.flattenConfig(inputs) }
func (u *Uniconf) flattenConfig(inputs []interface{}) (interface{}, error) {
	viper := viper.New()
	var yamlConfig = []byte(GetYAML())
	viper.SetConfigType("yaml")
	viper.ReadConfig(bytes.NewBuffer(yamlConfig))
	u.flatConfig = u.allSettings(viper)
	return nil, nil
}

func PrintConfig(inputs []interface{}) (interface{}, error) { return u.printConfig(inputs) }
func (u *Uniconf) printConfig(inputs []interface{}) (interface{}, error) {
	if len(inputs) > 0 {
		path := inputs[0].(string)
		fmt.Println(unitool.MarshallYaml(unitool.SearchMapWithPathStringPrefixes(u.config, path)))
	} else {
		fmt.Println(unitool.MarshallYaml(u.config))
	}
	return nil, nil
}

func processKeys(key string, source interface{}, parent interface{}, path string, phase *Phase, processors []*Processor, depth int, excludeKeys []string) {
	if depth > -100 {
		switch source.(type) {
		case string:
			for _, processor := range processors {
				if ((processor.IncludeKeys != nil && stringListContains(processor.IncludeKeys, key)) || processor.IncludeKeys == nil) &&
					((processor.ExcludeKeys != nil && !stringListContains(processor.ExcludeKeys, key)) || processor.ExcludeKeys == nil) {
					skip := false
					if processed, ok := parent.(map[string]interface{})[key+"_processed"]; ok {
						if unitool.StringListContains(processed.([]string), source.(string)) {
							skip = true
						}
					}
					if !skip {
						log.Debugf("processKeys path: %s, key: %s", path, key)
						value := source.(string)
						result, processed, mergeToParent, removeParentKey, replaceSource := processor.Callback(value, path, phase)
						if result != nil {
							result, _ = unitool.DeepCopyMap(result.(map[string]interface{}))
							if mergeToParent {
								unitool.Merge(parent, result)
							}
							if removeParentKey {
								log.Debugf("remove from list: %s", value)
								switch parent.(map[string]interface{})[key].(type) {
								case string:
									delete(parent.(map[string]interface{}), key)
								case []interface{}:
									parent.(map[string]interface{})[key] = unitool.RemoveFromList(parent.(map[string]interface{})[key].([]interface{}), value)
								}
							}
							if replaceSource != nil {
								value = replaceSource.(string)
							}
							if processed && removeParentKey {
								log.Debugf("Key processed: %s %v", path, value)
								if _, ok := parent.(map[string]interface{})[key+"_processed"]; !ok {
									parent.(map[string]interface{})[key+"_processed"] = make([]string, 0)
								}
								keyProcessed := source.(string) + " (" + value + ")"
								if source.(string) == value {
									keyProcessed = value
								}
								parent.(map[string]interface{})[key+"_processed"] = append(parent.(map[string]interface{})[key+"_processed"].([]string), keyProcessed)
							}
							if replaceSource != nil {
								source = replaceSource
							}
							if mergeToParent {
								parts := strings.Split(path, ".")
								p := strings.Join(parts[:len(parts)-1], ".")
								processKeys("", parent, source, p, phase, processors, depth, excludeKeys)
							}
						}
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
				//log.Debugf("processKeys() []interface{: %v", l)
				processKeys(key, l[i], parent, p, phase, processors, depth, excludeKeys)
			}
		case map[string]interface{}:
			for k, v := range source.(map[string]interface{}) {
				depth--
				if !stringListContains(excludeKeys, k) {
					processKeys(k, v, source, strings.Join([]string{path, k}, "."), phase, processors, depth, excludeKeys)
				} else {
					log.Debugf("Key skipped as excluded by parent: %s", k)
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

// ProcessKeys processes configuration.
func ProcessKeys(inputs []interface{}) (interface{}, error) { return u.processKeys(inputs) }
func (u *Uniconf) processKeys(inputs []interface{}) (interface{}, error) {
	path := inputs[0].(string)
	keys := strings.Split(path, ".")
	keyPrefix := inputs[1].(string)
	if keyPrefix != "" {
		keyPrefix = "." + keyPrefix
	}
	processors := inputs[2].([]*Processor)
	p := ""
	for _, v := range keys {
		p = strings.Trim(p+keyPrefix+"."+v, ".")
		var source interface{} = u.config
		if p != "" {
			source = unitool.SearchMapWithPathStringPrefixes(u.config, p)
		}
		processKeys("", source, nil, p, u.currentPhase, processors, 1, []string{keyPrefix})
	}
	return u.config, nil
}
