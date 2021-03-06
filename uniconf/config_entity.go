package uniconf

import (
	"errors"
	"strings"

	"github.com/aroq/uniconf/unitool"
	log "github.com/sirupsen/logrus"
)

type ConfigEntity struct {
	id     string
	title  string
	parent *ConfigEntity
	config map[string]interface{}
	source SourceHandler
}

func NewConfigEntity(s *Source, configMap map[string]interface{}) (*ConfigEntity, error) {
	if _, ok := configMap["config"]; !ok {
		if _, ok := configMap["stream"]; ok {
			stream := configMap["stream"].([]byte)
			if stream != nil {
				// TODO: check it.
				conf, err := unitool.UnmarshalByType(configMap["format"].(string), stream)
				if err == nil {
					configMap["config"] = conf
				}
			}
		}
	}
	if _, ok := configMap["config"]; !ok {
		return nil, errors.New("no ConfigEntity config is provided")
	}
	var parent *ConfigEntity
	if _, ok := configMap["parent"]; ok {
		parent = configMap["parent"].(*ConfigEntity)
	} else {
		parent = nil
	}

	if _, ok := configMap["title"]; !ok {
		configMap["title"] = configMap["id"]
	}
	c := &ConfigEntity{
		id:     configMap["id"].(string),
		title:  configMap["title"].(string),
		source: s,
		config: configMap["config"].(map[string]interface{}),
		parent: parent,
	}
	return c, nil
}

func (c *ConfigEntity) process() {
	if len(c.config) != 0 {
		c.processSources()
		c.processIncludes()
	}
}

func (c *ConfigEntity) processSources() {
	if sources, ok := c.config[sourceMapElementName].(map[string]interface{}); ok {
		for k, v := range sources {
			log.Printf("Process source: %s", k)
			if _, ok := u.sources[k]; !ok {
				// TODO: Check source type here.
				sourceType := "repo"
				switch v.(type) {
				case string:
					sourceType = "go-getter"
				case map[string]interface{}:
					if t, ok := v.(map[string]interface{})["type"]; ok {
						sourceType = t.(string)
					}
				}
				var source SourceHandler
				switch sourceType {
				case "go-getter":
					source = NewSourceGoGetter(k, v.(string))
				case "repo":
					source = NewSourceRepo(k, v.(map[string]interface{}))
				case "env":
					source = NewSourceEnv(k, v.(map[string]interface{}))
				case "file":
					source = NewSourceFile(k, v.(map[string]interface{}))
				case "config_map":
					source = NewSourceConfigMap(k, v.(map[string]interface{}))
				default:
					source = NewSourceRepo(k, v.(map[string]interface{}))
				}
				u.sources[k] = source

				if autoloadID := source.Autoload(); autoloadID != "" {
					if _, ok := c.config[IncludeListElementName]; !ok {
						c.config[IncludeListElementName] = make([]interface{}, 0)
					}
					c.config[IncludeListElementName] = append(c.config[IncludeListElementName].([]interface{}), strings.Join([]string{source.Name(), autoloadID}, ":"))
				}
			} else {
				log.Printf("Source: %s already loaded", k)
			}
		}
	}
}

func (c *ConfigEntity) processIncludes() {
	parseScenario := func(scenario string) (sourceName, scenarioName string) {
		sourceName, include := "", ""
		if strings.Contains(scenario, ":") {
			s := strings.Split(scenario, ":")
			sourceName, include = s[0], s[1]
		} else {
			sourceName, include = c.source.Name(), scenario
		}
		return sourceName, include
	}

	includesConfig := make(map[string]interface{})

	if includes, ok := c.config[IncludeListElementName]; ok {
		for _, v := range includes.([]interface{}) {
			sourceName, scenarioID := parseScenario(v.(string))
			// TODO: check if title is needed.
			title := scenarioID
			source := u.getSource(sourceName)
			ids, _ := source.GetIncludeConfigEntityIds(scenarioID)
			for _, id := range ids {
				log.Printf("Process include: %s", source.Path()+":"+id)
				if subConfigEntity, err := source.LoadConfigEntity(map[string]interface{}{"id": id, "title": title, "parent": c}); err == nil {
					unitool.Merge(includesConfig, subConfigEntity.config, true)
				} else {
					log.Warnf("LoadConfigEntity error: %v", err)
				}
			}
		}
		c.config["from_processed"] = includes
		delete(c.config, IncludeListElementName)
	}
	if c.config != nil {
		unitool.Merge(includesConfig, c.config, true)
		c.config = includesConfig
	}
}
