package uniconf

import (
	"github.com/aroq/uniconf/unitool"
	log "github.com/sirupsen/logrus"
	"strings"
)

type Processor struct {
	IncludeKeys []string
	ExcludeKeys []string
	Callback    func(source interface{}, path string, phase *Phase) (result interface{}, processed, mergeToParent, removeParentKey bool, replaceSource interface{})
}

var processedFromKeys = make(map[string]interface{})

func InterpolateProcess(source interface{}, path string, phase *Phase) (result interface{}, processed, mergeToParent, removeParentKey bool, replaceSource interface{}) {
	if strings.Contains(source.(string), "${") {
		s := InterpolateString(source.(string), u.flatConfig)
		return s, true, false, false, s
	} else {
		return nil, false, false, false, nil
	}
}

func FromProcess(source interface{}, path string, phase *Phase) (result interface{}, processed, mergeToParent, removeParentKey bool, replaceSource interface{}) {
	from := InterpolateString(source.(string), u.flatConfig)
	if result, ok := processedFromKeys[from]; ok {
		return result, true, true, true, from
	}

	processorParams, err := unitool.CollectKeyParamsFromJsonPath(u.config, from, "processors")
	if err != nil {
		log.Errorf("Error: %v", err)
	}
	if processorParams != nil {
		fromMode := unitool.SearchMapWithPathStringPrefixes(processorParams, "from.mode")
		if fromMode != nil {
			modeParam := fromMode.(string)
			phaseName := phaseFullName(phase)
			if modeParam != "" && strings.HasPrefix(phaseName, modeParam) {
				result, err := unitool.CollectKeyParamsFromJsonPath(u.config, from, "params")
				if err != nil {
					log.Errorf("Error: %v", err)
				}
				processedFromKeys[from] = result
				return result, true, true, true, from
			}
		}
	}
	return nil, false, false, false, nil
}

