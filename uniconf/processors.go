package uniconf

import (
	"github.com/aroq/uniconf/unitool"
	log "github.com/sirupsen/logrus"
	"strings"
)

// Process processes configuration.
func Process(source interface{}, path, phase string) interface{} {
	processors := []map[string]interface{}{
		{
			"id":          "fromProcessor",
			"include_key": "from",
		},
	}
	for i := 0; i < len(processors); i++ {
		if processors[i]["id"] == "fromProcessor" {
			fromProcess(source, path, phase)
		}
	}
	return source
}

func fromProcess(source interface{}, path, phase string) {
	processFromFunc := func(from string) (processed bool) {
		processed = false
		processorParams, err := unitool.CollectKeyParamsFromJsonPath(u.config, from, "processors")
		if err != nil {
			log.Errorf("Error: %v", err)
		}
		if processorParams != nil {
			fromMode := unitool.SearchMapWithPathStringPrefixes(processorParams, "from.mode")
			if fromMode != nil {
				modeParam := fromMode.(string)
				if modeParam != "" && modeParam == phase {
					result, err := unitool.CollectKeyParamsFromJsonPath(u.config, from, "params")
					if err != nil {
						log.Errorf("Error: %v", err)
					}
					fromProcess(result, path, phase)
					unitool.Merge(source.(map[string]interface{}), result)
					processed = true
				}
			}
		}
		return
	}

	switch source.(type) {
	case map[string]interface{}:
		for k, v := range source.(map[string]interface{}) {
			switch v.(type) {
			case map[string]interface{}:
				fromProcess(v, strings.Join([]string{path, k}, "."), phase)
			case string:
				if k == "from" {
					processed := processFromFunc(v.(string))
					if processed {
						delete(source.(map[string]interface{}), "from")
						source.(map[string]interface{})["from_processed"] = v
					}
				}
			case []interface{}:
				l := v.([]interface{})
				processed := false
				for i := 0; i < len(l); i++ {
					if k == "from" {
						p := processFromFunc(l[i].(string))
						if p {
							processed = true
						}
					} else {
						fromProcess(l[i], strings.Join([]string{path, k}, "."), phase)
					}
				}
				if processed {
					delete(source.(map[string]interface{}), "from")
					source.(map[string]interface{})["from_processed"] = v
				}
			}
		}
	}
}

