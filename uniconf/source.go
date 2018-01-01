package uniconf

import (
	"github.com/aroq/uniconf/unitools"
	"path"
)

type Source struct {
	name      string
	repo      string
	ref       string
	refPrefix string
	path      string
	isLoaded  bool
}

const (
	refPrefix     = "refs/"
	refHeadPrefix = refPrefix + "heads/"
	//refTagPrefix    = refPrefix + "tags/"
)

func (s *Source) LoadSource() error {
	err := unitools.GitClone(s.repo, s.refPrefix+s.ref, s.path, 1, true)
	if err == nil {
		s.isLoaded = true
	}
	return err
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
	return &Source{name: sourceName, path: path, repo: sourceMap["repo"].(string), ref: ref, refPrefix: prefix.(string), isLoaded: false}
}
