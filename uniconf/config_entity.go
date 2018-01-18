package uniconf

import (
	"errors"
	"github.com/aroq/uniconf/unitool"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

type ConfigEntity struct {
	id     string
	title  string
	parent *ConfigEntity
	config  map[string]interface{}
	source  SourceHandler
	history map[string]interface{}
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
		id:      configMap["id"].(string),
		title:   configMap["title"].(string),
		source:  s,
		config:  configMap["config"].(map[string]interface{}),
		history: make(map[string]interface{}),
		parent:  parent,
	}
	c.history = make(map[string]interface{})
	c.saveHistory("", c.config)
	return c, nil
}

func (c *ConfigEntity) getHistoryChain() string {
	name := ""
	if c.parent != nil {
		name += c.parent.getHistoryChain() + " -> "
	}
	name += c.source.Name() + ":" + c.title + " (" + c.id + ")"
	return name
}

func (c *ConfigEntity) saveHistory(path string, config map[string]interface{}) {
	saver := func(historyKey string, v interface{}) {
		if _, ok := c.history[historyKey]; !ok {
			c.history[historyKey] = make(map[string]interface{})
			c.history[historyKey].(map[string]interface{})["load"] = make(map[string]interface{})
			c.history[historyKey].(map[string]interface{})["order"] = make([]string, 0)
		}
		if v == nil {
			v = "..."
		}
		newEntry := map[string]interface{}{c.source.Name() + ":" + c.id: v}
		c.history[historyKey].(map[string]interface{})["load"] = unitool.Merge(c.history[historyKey].(map[string]interface{})["load"], newEntry)
		history := c.getHistoryChain()
		c.history[historyKey].(map[string]interface{})["order"] = append(c.history[historyKey].(map[string]interface{})["order"].([]string), history)
	}

	for k, v := range config {
		historyKey := strings.Trim(strings.Join([]string{path, k}, "."), ".")
		switch v.(type) {
		case map[string]interface{}:
			saver(historyKey, nil)
			c.saveHistory(historyKey, v.(map[string]interface{}))
		case []interface{}:
			for lk, lv := range v.([]interface{}) {
				switch lv.(type) {
				case map[string]interface{}:
					saver(historyKey, nil)
					c.saveHistory(historyKey+"["+strconv.Itoa(lk)+"]", lv.(map[string]interface{}))
				}
			}
		case string, int, bool:
			saver(historyKey, v)
		}
	}
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
				if t, ok := v.(map[string]interface{})["type"]; ok {
					sourceType = t.(string)
				}
				var source SourceHandler
				switch sourceType {
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

				if autoloadId := source.Autoload(); autoloadId != "" {
					if _, ok := c.config[includeListElementName]; !ok {
						c.config[includeListElementName] = make([]interface{}, 0)
					}
					c.config[includeListElementName] = append(c.config[includeListElementName].([]interface{}), strings.Join([]string{source.Name(), autoloadId}, ":"))
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
	history := make(map[string]interface{})

	if includes, ok := c.config[includeListElementName]; ok {
		for _, v := range includes.([]interface{}) {
			sourceName, scenarioId := parseScenario(v.(string))
			// TODO: check if title is needed.
			title := scenarioId
			source := u.getSource(sourceName)
			ids, _ := source.GetIncludeConfigEntityIds(scenarioId)
			for _, id := range ids {
				log.Printf("Process include: %s", source.Path() + ":" + id)
				if subConfigEntity, err := source.LoadConfigEntity(map[string]interface{}{"id": id, "title": title, "parent": c}); err == nil {
					unitool.Merge(includesConfig, subConfigEntity.config)
					unitool.Merge(history, subConfigEntity.history)
				} else {
					log.Warnf("LoadConfigEntity error: %v", err)
				}
			}
		}
		c.config["from_processed"] = includes
		delete(c.config, includeListElementName)
	}
	if c.config != nil {
		unitool.Merge(includesConfig, c.config)
		unitool.Merge(c.history, history)
		c.config = includesConfig
	}
}
