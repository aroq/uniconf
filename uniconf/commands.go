package uniconf

import (
	"github.com/aroq/uniconf/unitool"
	"fmt"
	"strings"
)

func Collect(jsonPath, key string) string { return u.collect(jsonPath, key) }
func (u *Uniconf) collect(jsonPath, key string) string {
	result, _ := unitool.CollectKeyParamsFromJsonPath(u.config, jsonPath, key)
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

func GetYaml() (yamlString string) { return u.getYaml() }
func (u *Uniconf) getYaml() string {
	return unitool.MarshallYaml(u.config)
}

func GetJson() (yamlString string) { return u.getJson() }
func (u *Uniconf) getJson() string {
	return unitool.MarshallJson(u.config)
}

