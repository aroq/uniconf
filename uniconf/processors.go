package uniconf

import (
	"github.com/aroq/uniconf/unitool"
	log "github.com/sirupsen/logrus"
	"strings"
	"regexp"
	"github.com/hashicorp/hil"
	"github.com/hashicorp/hil/ast"
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

func InterpolateString(input string, config map[string]interface{}) string {
	if config == nil {
		config = u.flatConfig
	}
	if strings.Contains(input, "${") {
		r, _ := regexp.Compile("(.*\\$\\{)(context\\.)(.*\\})")
		input = r.ReplaceAllString(input, "$1$3")

		tree, err := hil.Parse(input)
		if err != nil {
			log.Fatal(err)
		}

		deepGet := ast.Function{
			ArgTypes:   []ast.Type{ast.TypeString},
			ReturnType: ast.TypeString,
			Variadic:   false,
			Callback: func(inputs []interface{}) (interface{}, error) {
				input := inputs[0].(string)
				return unitool.SearchMapWithPathStringPrefixes(config, input), nil
			},
		}

		configMap := map[string]ast.Variable{}
		for k, v := range config {
			configMap[k], _ = hil.InterfaceToVariable(v)
		}

		c := &hil.EvalConfig{
			GlobalScope: &ast.BasicScope{
				VarMap: configMap,
				FuncMap: map[string]ast.Function{
					"deepGet": deepGet,
				},
			},
		}

		result, err := hil.Eval(tree, c)
		if err != nil {
			log.Fatal(err)
		}

		//fmt.Printf("Type: %s\n", result.Type)
		//fmt.Printf("Value: %s\n", result.Value)

		return result.Value.(string)
	} else {
		return input
	}
}
