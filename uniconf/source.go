package uniconf

import (
	"errors"
	"fmt"
	"github.com/aroq/uniconf/unitool"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"strings"
)

type SourceHandler interface {
	Name() string
	Path() string
	Autoload() string
	LoadSource() error
	IsLoaded() bool
	GetIncludeConfigEntityIds(scenarioId string) ([]string, error)
	LoadConfigEntity(configMap map[string]interface{}) (*ConfigEntity, error)
	ConfigEntity(id string) (*ConfigEntity, bool)
}

type Source struct {
	name           string
	isLoaded       bool
	configEntities map[string]*ConfigEntity
	autoloadId     string
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

type SourceConfigMap struct {
	Source
	configMap map[string]interface{}
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

func (s *Source) GetIncludeConfigEntityIds(scenarioId string) ([]string, error) {
	return []string{scenarioId}, nil
}

func (s *Source) Autoload() string {
	return s.autoloadId
}

func (s *Source) LoadConfigEntity(configMap map[string]interface{}) (*ConfigEntity, error) {
	//fmt.Println("Process %s: %s", configMap["name"], configMap["id"])
	if c, ok := s.ConfigEntity(configMap["id"].(string)); ok {
		return c, nil
	} else {
		c, err := NewConfigEntity(s, configMap)
		if err != nil {
			log.Fatalf("Error creating ConfigEntity: %v", err)
		}
		s.configEntities[c.id] = c
		c.process()
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

func (s *SourceFile) GetIncludeConfigEntityIds(scenarioId string) ([]string, error) {
	ids := make([]string, 0)
	files := make([]string, 0)
	if strings.Index(scenarioId, "/") == 0 {
		ids = append(ids, scenarioId)
	} else {
		if strings.Contains(scenarioId, "/") {
			s := strings.Split(scenarioId, "/")
			id := ""
			for _, v := range s {
				if v != "" {
					if id == "" {
						id += v
					} else {
						id += "/" + v
					}
					ids = append(ids, id)
				}
			}
		} else {
			ids = append(ids, scenarioId)
		}
	}
	var includeFileNamesToCheck []string
	for _, id := range ids {
		if !(strings.Index(scenarioId, "/") == 0) {
			scenarioId = path.Join(includesPath, id)
			includeFileName := path.Join(s.Path(), includesPath, id)
			includeFileNamesToCheck = append(includeFileNamesToCheck, includeFileName+".yaml", includeFileName+".yml", includeFileName+".json", path.Join(includeFileName, mainConfigFileName))
		} else {
			scenarioId = strings.Trim(id, "/")
			includeFileName := path.Join(s.Path(), scenarioId)
			includeFileNamesToCheck = append(includeFileNamesToCheck, includeFileName)
		}
	}

	for _, f := range includeFileNamesToCheck {
		if _, err := os.Stat(f); err == nil {
			files = append(files, f)
		}
	}

	return files, nil
}

func (s *SourceFile) LoadConfigEntity(configMap map[string]interface{}) (*ConfigEntity, error) {
	//fmt.Printf("Process %s: %s", configMap["name"], configMap["id"])
	if _, ok := s.ConfigEntity(configMap["id"].(string)); !ok {
		if scenarioId, ok := configMap["id"].(string); ok {
			stream := unitool.ReadFile(scenarioId)
			configMap["stream"] = stream
			if _, ok := configMap["format"]; !ok {
				configMap["format"] = unitool.FormatByExtension(scenarioId)
			}
			conf, err := unitool.UnmarshalByType(configMap["format"].(string), stream)
			if err == nil {
				configMap["config"] = conf
				if configEntity, err := s.Source.LoadConfigEntity(configMap); err == nil {
					return configEntity, nil
				} else {
					log.Warnf("LoadConfigEntity error: %v", err)
				}
			} else {
				log.Errorf("UnmarshalByType error: %v", err)
			}
		} else {
			log.Errorf("Config map doesn't contain id")
		}
	} else {
		return nil, errors.New(fmt.Sprintf("Config entity already loaded: %s", configMap["id"].(string)))
	}
	return nil, errors.New(fmt.Sprintf("file %s doesnt'exists", configMap["id"].(string)))
}

func (s *SourceEnv) GetIncludeConfigEntityIds(scenarioId string) ([]string, error) {
	ids := make([]string, 0)
	envVars := make([]string, 0)
	if strings.Contains(scenarioId, "_") {
		parts := strings.Split(scenarioId, "_")
		id := ""
		for _, v := range parts {
			if v != "" {
				if id == "" {
					id += v
				} else {
					id += "_" + v
				}
				ids = append(ids, id)
			}
		}
	} else {
		ids = append(ids, scenarioId)
	}
	for _, v := range ids {
		if _, ok := os.LookupEnv(v); ok {
			envVars = append(envVars, v)
		}
	}
	return envVars, nil
}

func (s *SourceEnv) LoadConfigEntity(configMap map[string]interface{}) (*ConfigEntity, error) {
	log.Info(fmt.Sprintf("Process %s: %s", configMap["name"], configMap["id"]))
	if value, ok := os.LookupEnv(configMap["id"].(string)); ok {
		configMap["stream"] = []byte(value)
		if _, ok := configMap["format"]; !ok {
			configMap["format"] = "json"
		}
		return s.Source.LoadConfigEntity(configMap)
	}
	return nil, errors.New(fmt.Sprintf("environment variable %s doesnt'exists", configMap["id"].(string)))
}

func (s *SourceConfigMap) LoadConfigEntity(configMap map[string]interface{}) (*ConfigEntity, error) {
	//fmt.Printf("Process %s: %s", configMap["name"], configMap["id"])
	if _, ok := s.ConfigEntity(configMap["id"].(string)); !ok {
		if value, ok := s.configMap[configMap["id"].(string)]; ok {
			switch value.(type) {
			case map[string]interface{}:
				configMap["config"] = value
			case []byte:
				// TODO: check if JSON format is needed at all here.
				format := "yaml"
				configMap["config"], _ = unitool.UnmarshalByType(format, value.([]byte))
			}
			return s.Source.LoadConfigEntity(configMap)
		}
	} else {
		return nil, errors.New(fmt.Sprintf("config entity already loaded: %s", configMap["id"].(string)))
	}
	return nil, errors.New(fmt.Sprintf("source config map entry %s doesnt'exists", configMap["id"].(string)))
}

func (s *SourceConfigMap) GetIncludeConfigEntityIds(scenarioId string) ([]string, error) {
	ids := make([]string, 0)
	files := make([]string, 0)
	if strings.Index(scenarioId, "/") == 0 {
		ids = append(ids, scenarioId)
	} else {
		if strings.Contains(scenarioId, "/") {
			s := strings.Split(scenarioId, "/")
			id := ""
			for _, v := range s {
				if v != "" {
					if id == "" {
						id += v
					} else {
						id += "/" + v
					}
					ids = append(ids, id)
				}
			}
		} else {
			ids = append(ids, scenarioId)
		}
	}

	for _, f := range ids {
		if _, ok := s.configMap[f]; ok {
			files = append(files, f)
		}
	}

	return files, nil
}

func NewSource(sourceName string, sourceMap map[string]interface{}) *Source {
	source := &Source{
		name:           sourceName,
		isLoaded:       false,
		configEntities: make(map[string]*ConfigEntity),
	}
	if autoloadId, ok := sourceMap["autoload"]; ok {
		source.autoloadId = autoloadId.(string)
	}
	return source
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
		repo:       sourceMap["repo"].(string),
		ref:        ref,
		refPrefix:  prefix.(string),
	}
}

func NewSourceFile(sourceName string, sourceMap map[string]interface{}) *SourceFile {
	return &SourceFile{
		Source: *NewSource(sourceName, sourceMap),
		path:   sourceMap["path"].(string),
	}
}

func NewSourceEnv(sourceName string, sourceMap map[string]interface{}) *SourceEnv {
	return &SourceEnv{
		Source: *NewSource(sourceName, sourceMap),
	}
}

func NewSourceConfigMap(sourceName string, sourceMap map[string]interface{}) *SourceConfigMap {
	return &SourceConfigMap{
		Source:    *NewSource(sourceName, sourceMap),
		configMap: sourceMap["configMap"].(map[string]interface{}),
	}
}
