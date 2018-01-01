package uniconf

import (
	"encoding/json"
	"github.com/aroq/uniconf/unitools"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"strings"
	"fmt"
)

type ConfigEntity struct {
	name       string
	stream     []byte
	config     map[string]interface{}
	format     string
	configType string
	source     *Source
}

func (c *ConfigEntity) processSources() {
	if sources, ok := c.config[sourceMapElementName].(map[string]interface{}); ok {
		for k, v := range sources {
			log.Printf("Process source: %s", k)
			if _, ok := u.sources[k]; !ok {
				source := NewSource(k, v.(map[string]interface{}))
				u.sources[k] = source
			} else {
				log.Printf("Source: %s already loaded", k)
			}
		}
	}
}

func (c *ConfigEntity) processIncludes() map[string]interface{} {
	includesConfig := make(map[string]interface{})

	if includes, ok := c.config[includeListElementName]; ok {
		for _, include := range includes.([]interface{}) {
			scenario := include.(string)
			log.Printf("Process include: %s", scenario)
			sourceName, include := "", ""
			if strings.Contains(scenario, ":") {
				s := strings.Split(scenario, ":")
				sourceName, include = s[0], s[1]
			} else {
				sourceName, include = c.source.name, scenario
			}
			if !(strings.Index(include, "/") == 0) {
				include = path.Join(includesPath, include)
			}

			source := u.getSource(sourceName)

			var includeFileNamesToCheck []string
			includeFileName := path.Join(source.path, include)

			includeFileNamesToCheck = append(includeFileNamesToCheck, includeFileName+".yaml")
			includeFileNamesToCheck = append(includeFileNamesToCheck, includeFileName+".yml")
			includeFileNamesToCheck = append(includeFileNamesToCheck, path.Join(includeFileName, mainConfigFileName))

			for _, f := range includeFileNamesToCheck {
				if _, err := os.Stat(f); err == nil {
					subConfigEntity := loadConfigEntity(map[string]interface{}{"name": f, "configType": "file", "source": source, "id": f, "format": "yaml"})
					unitools.Merge(includesConfig, subConfigEntity.config)
				}
			}
		}
	}
	return includesConfig
}

func (c *ConfigEntity) processAfterLoad() {
	c.processSources()
	includesConfig := c.processIncludes()
	unitools.Merge(includesConfig, c.config)
	c.config = includesConfig
}

func loadConfigEntity(configMap map[string]interface{}) (*ConfigEntity) {
	fmt.Sprintf("Process %s %s: %s", configMap["configType"], configMap["name"], configMap["id"])

	stream := make([]byte, 0)
	if configMap["configType"] == "file" {
		stream = unitools.ReadFile(configMap["id"].(string))
	} else if configMap["configType"] == "env_var" {
		stream =[]byte(os.Getenv(configMap["id"].(string)))
	}

	// TODO: return error.
	if stream == nil {
		return nil
	}

	c := &ConfigEntity{name: configMap["name"].(string), source: configMap["source"].(*Source), stream: stream, format: configMap["format"].(string)}
	c.config = make(map[string]interface{})

	if c.format == "yaml" {
		c.config, _ = unitools.UnmarshalYaml(c.stream)
	} else if c.format == "json" {
		json.Unmarshal(c.stream, &c.config)
	}
	if len(c.config) != 0 {
		c.config["format"] = c.format
		c.processAfterLoad()
		return c
	}
	// TODO: return error.
	return nil
}
