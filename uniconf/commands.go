package uniconf

import (
	"fmt"
	"strings"

	"github.com/aroq/uniconf/unitool"
)

func Collect(jsonPath, key string) string { return u.collect(jsonPath, key) }
func (u *Uniconf) collect(jsonPath, key string) string {
	result, _ := unitool.DeepCollectParams(u.config, jsonPath, key)
	return unitool.MarshallYaml(result)
}

func Explain(jsonPath, key string) { u.explain(jsonPath, key) }
func (u *Uniconf) explain(jsonPath, key string) {
	result := unitool.SearchMapWithPathStringPrefixes(u.config, jsonPath)
	fmt.Println("Result:")
	fmt.Println(unitool.MarshallYaml(result))

	if history, ok := u.config["history"].(map[string]interface{})[strings.Trim(jsonPath, ".")]; ok {
		fmt.Println("Load history:")
		fmt.Println(unitool.MarshallYaml(history))
	}

	//Process(result, "", "config")
	//fmt.Println("From processed result:")
	//fmt.Println(unitool.MarshallYaml(result))
}

func GetYAML() (yamlString string) { return u.getYAML() }
func (u *Uniconf) getYAML() string {
	return unitool.MarshallYaml(u.config)
}

func GetJSON() (yamlString string) { return u.getJSON() }
func (u *Uniconf) getJSON() string {
	return unitool.MarshallJSON(u.config)
}
