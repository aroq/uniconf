package uniconf

import (
	"github.com/aroq/uniconf/unitool"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"strconv"
	"strings"
)

type ConfigEntity struct {
	id      string
	title  string
	parent  *ConfigEntity
	stream  []byte
	config  map[string]interface{}
	format  string
	source  SourceHandler
	history map[string]interface{}
}

func (c *ConfigEntity) Read() error {
	conf, err := unitool.UnmarshalByType(c.format, c.stream)
	if err == nil {
		c.config = conf
		c.history = make(map[string]interface{})
		c.saveHistory("", c.config)
	}
	return err
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

func (c *ConfigEntity) Process() {
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
				// TODO: Check repo type here.
				source := NewSourceRepo(k, v.(map[string]interface{}))
				u.sources[k] = source
			} else {
				log.Printf("Source: %s already loaded", k)
			}
		}
	}
}

func (c *ConfigEntity) processIncludes() {
	includesConfig := make(map[string]interface{})
	history := make(map[string]interface{})

	if includes, ok := c.config[includeListElementName]; ok {
		for _, include := range includes.([]interface{}) {
			scenario := include.(string)
			log.Printf("Process include: %s", scenario)
			sourceName, include := "", ""
			if strings.Contains(scenario, ":") {
				s := strings.Split(scenario, ":")
				sourceName, include = s[0], s[1]
			} else {
				sourceName, include = c.source.Name(), scenario
			}
			title := include
			if !(strings.Index(include, "/") == 0) {
				include = path.Join(includesPath, include)
			}
			source := u.getSource(sourceName)
			var includeFileNamesToCheck []string
			includeFileName := path.Join(source.Path(), include)
			includeFileNamesToCheck = append(includeFileNamesToCheck, includeFileName+".yaml", includeFileName+".yml", includeFileName+".json", path.Join(includeFileName, mainConfigFileName))

			for _, f := range includeFileNamesToCheck {
				if _, err := os.Stat(f); err == nil {
					if _, ok := source.ConfigEntity(f); !ok {
						if subConfigEntity, err := source.LoadConfigEntity(map[string]interface{}{"id": f, "title": title, "parent": c}); err == nil {
							unitool.Merge(includesConfig, subConfigEntity.config)
							unitool.Merge(history, subConfigEntity.history)
						}
					}
				}
			}
		}
		c.config["from_processed"] = includes
		delete(c.config, includeListElementName)
	}
	unitool.Merge(includesConfig, c.config)
	unitool.Merge(c.history, history)
	c.config = includesConfig
}
