package uniconf

import (
	"github.com/aroq/uniconf/unitool"
)

func Collect(jsonPath, key string) string { return u.collect(jsonPath, key) }
func (u *Uniconf) collect(jsonPath, key string) string {
	result, _ := unitool.DeepCollectParams(u.config, jsonPath, key)
	return unitool.MarshallYaml(result)
}

func GetYAML() (yamlString string) { return u.getYAML() }
func (u *Uniconf) getYAML() string {
	return unitool.MarshallYaml(u.config)
}

func GetJSON() (yamlString string) { return u.getJSON() }
func (u *Uniconf) getJSON() string {
	return unitool.MarshallJSON(u.config)
}
