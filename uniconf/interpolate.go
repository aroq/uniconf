package uniconf

import (
	"github.com/hashicorp/hil"
	"github.com/hashicorp/hil/ast"
	"log"
	"github.com/aroq/uniconf/unitool"
	"regexp"
	"strings"
)

func InterpolateString(input string, config map[string]interface{}) string {
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
