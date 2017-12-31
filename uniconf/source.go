package uniconf

import (
	"path"
	"github.com/aroq/uniconf/unitools"
)

type Source struct {
	Repo         string
	Ref          string
	RefPrefix    string
	Path         string
}

func (s *Source) LoadSource() error {
	//log.Printf("Cloning source repo: %s, ref: %s\n", s.Repo, s.Ref)
	return unitools.GitClone(s.Repo, s.RefPrefix + s.Ref, s.Path, 1, true)
}

func NewSource(sourceName string, sourceMap map[string]interface{}) *Source {
	path := path.Join(appTempFilesPath, sourcesStoragePath, sourceName)
	var ref string
	if _, ok := sourceMap["ref"]; ok {
		ref = sourceMap["ref"].(string)
	} else {
		ref = "master"
	}
	prefix, ok := sourceMap["prefix"]
	if !ok {
		prefix = refHeadPrefix
	}
	return &Source{Path: path, Repo: sourceMap["repo"].(string), Ref: ref, RefPrefix: prefix.(string)}
}
