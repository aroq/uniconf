package uniconf

import (
	"github.com/aroq/uniconf/unitool"
	"path"
	"fmt"
	"os"
	"path/filepath"
	"errors"
)

type SourceHandler interface {
	Name() string
	Path() string
	LoadSource() error
	IsLoaded() bool
	LoadConfigEntity(configMap map[string]interface{}) (*ConfigEntity, error)
	ConfigEntity(id string) (*ConfigEntity, bool)
}

type Source struct {
	name     string
	isLoaded bool
	configEntities map[string]*ConfigEntity
}

type SourceFile struct {
	Source
	path string
}

type SourceRepo struct {
	SourceFile
	repo      string
	ref       string
	refPrefix string
}

type SourceEnv struct {
	Source
}

const (
	refPrefix     = "refs/"
	refHeadPrefix = refPrefix + "heads/"
	//refTagPrefix    = refPrefix + "tags/"
)

func (s *Source) Name() string {
	return s.name
}

func (s *Source) Path() string {
	return ""
}

func (s *Source) IsLoaded() bool {
	return s.isLoaded
}

func (s *Source) LoadSource() error {
	s.isLoaded = true
	return nil
}

func (s *Source) LoadConfigEntity(configMap map[string]interface{}) (*ConfigEntity, error) {
	fmt.Sprintf("Process %s: %s", configMap["name"], configMap["id"])
	if c, ok := s.ConfigEntity(configMap["id"].(string)); ok {
		return c, nil
	} else {
		stream := configMap["stream"].([]byte)
		if stream == nil {
			return nil, errors.New("empty config stream")
		}
		var parent *ConfigEntity
		if _, ok := configMap["parent"]; ok {
			parent = configMap["parent"].(*ConfigEntity)
		} else {
			parent = nil
		}

		if _, ok := configMap["title"]; !ok {
			configMap["title"] = configMap["id"]
		}

		c := &ConfigEntity{id: configMap["id"].(string), title: configMap["title"].(string), source: s, stream: stream, format: configMap["format"].(string), parent: parent}
		s.configEntities[c.id] = c
		c.Read()
		c.Process()
		return c, nil
	}
}

func (s *Source) ConfigEntity(id string) (*ConfigEntity, bool) {
	if c, ok := s.configEntities[id]; ok {
		return c, true
	} else {
		return nil, false
	}
}

func (s *SourceFile) Path() string {
	return s.path
}

func (s *SourceRepo) LoadSource() error {
	err := unitool.GitClone(s.repo, s.refPrefix+s.ref, s.path, 1, true)
	if err == nil {
		s.Source.LoadSource()
	}
	return err
}

func (s *SourceFile) LoadConfigEntity(configMap map[string]interface{}) (*ConfigEntity, error) {
	file := configMap["id"].(string)
	if _, err := os.Stat(file); err == nil {
		configMap["stream"] = unitool.ReadFile(file)
		if _, ok := configMap["format"]; !ok {
			extension := filepath.Ext(file)
			switch extension {
			case ".yaml", ".yml":
				configMap["format"] = "yaml"
			case ".json":
				configMap["format"] = "json"
			}
		}
		return s.Source.LoadConfigEntity(configMap)
	}
	return nil, errors.New(fmt.Sprintf("file %s doesnt'exists", configMap["id"].(string)))
}

func (s *SourceEnv) LoadConfigEntity(configMap map[string]interface{}) (*ConfigEntity, error) {
	if value, ok := os.LookupEnv(configMap["id"].(string)); ok {
		configMap["stream"] = []byte(value)
		if _, ok := configMap["format"]; !ok {
			configMap["format"] = "json"
		}
		return s.Source.LoadConfigEntity(configMap)
	}
	return nil, errors.New(fmt.Sprintf("environment variable %s doesnt'exists", configMap["id"].(string)))
}

func NewSource(sourceName string, sourceMap map[string]interface{}) *Source {
	return &Source{
		name:     sourceName,
		isLoaded: false,
		configEntities: make(map[string]*ConfigEntity),
	}
}

func NewSourceRepo(sourceName string, sourceMap map[string]interface{}) *SourceRepo {
	sourceMap["path"] = path.Join(appTempFilesPath, sourcesStoragePath, sourceName)
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
	return &SourceRepo{
		SourceFile: *NewSourceFile(sourceName, sourceMap),
		repo:      sourceMap["repo"].(string),
		ref:       ref,
		refPrefix: prefix.(string),
	}
}

func NewSourceFile(sourceName string, sourceMap map[string]interface{}) *SourceFile {
	return &SourceFile{
		Source: *NewSource(sourceName, sourceMap),
		path: sourceMap["path"].(string),
	}
}

func NewSourceEnv(sourceName string, sourceMap map[string]interface{}) *SourceEnv {
	return &SourceEnv{
		Source: *NewSource(sourceName, sourceMap),
	}
}

