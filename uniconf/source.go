package uniconf

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/aroq/uniconf/unitool"
	"github.com/hashicorp/go-getter"
	log "github.com/sirupsen/logrus"
)

type SourceHandler interface {
	Name() string
	Path() string
	Autoload() string
	LoadSource() error
	IsLoaded() bool
	GetIncludeConfigEntityIds(scenarioID string) ([]string, error)
	LoadConfigEntity(configMap map[string]interface{}) (*ConfigEntity, error)
	ConfigEntity(id string) (*ConfigEntity, bool)
}

type Source struct {
	name           string
	src            string
	dst            string
	isLoaded       bool
	configEntities map[string]*ConfigEntity
	autoloadID     string
}

type SourceFile struct {
	Source
	path string
}

type SourceGoGetter struct {
	SourceFile
	url string
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

func (s *Source) GetIncludeConfigEntityIds(scenarioID string) ([]string, error) {
	return []string{scenarioID}, nil
}

func (s *Source) Autoload() string {
	return s.autoloadID
}

func (s *Source) LoadConfigEntity(configMap map[string]interface{}) (*ConfigEntity, error) {
	//fmt.Println("Process %s: %s", configMap["name"], configMap["id"])
	if c, ok := s.ConfigEntity(configMap["id"].(string)); ok {
		return c, nil
	}

	c, err := NewConfigEntity(s, configMap)
	if err != nil {
		log.Fatalf("Error creating ConfigEntity: %v", err)
	}
	s.configEntities[c.id] = c
	c.process()
	return c, nil
}

func (s *Source) ConfigEntity(id string) (*ConfigEntity, bool) {
	if c, ok := s.configEntities[id]; ok {
		return c, true
	}

	return nil, false
}

func (s *SourceFile) Path() string {
	return s.path
}

func (s *SourceGoGetter) LoadSource() error {
	err := getter.GetAny(s.path, s.url)
	if err == nil {
		err = s.Source.LoadSource()
	}
	return err
}

func (s *SourceRepo) LoadSource() error {
	err := unitool.GitClone(s.repo, s.refPrefix+s.ref, s.path, 1, true)
	if err == nil {
		err = s.Source.LoadSource()
	}
	return err
}

func (s *SourceFile) GetIncludeConfigEntityIds(id string) ([]string, error) {
	ids := make([]string, 0)
	files := make([]string, 0)
	ids = append(ids, id)

	//TODO: Refactor this logic as only single is processed now
	for _, id := range ids {
		id = strings.Trim(id, "/")
		fileName := path.Join(s.Path(), id)
		if _, err := os.Stat(fileName); err == nil {
			files = append(files, fileName)
		}
	}

	return files, nil
}

func (s *SourceFile) LoadConfigEntity(configMap map[string]interface{}) (*ConfigEntity, error) {
	//fmt.Printf("Process %s: %s", configMap["name"], configMap["id"])
	if _, ok := s.ConfigEntity(configMap["id"].(string)); !ok {
		if scenarioID, ok := configMap["id"].(string); ok {
			stream := unitool.ReadFile(scenarioID)
			configMap["stream"] = stream
			if _, ok := configMap["format"]; !ok {
				configMap["format"] = unitool.FormatByExtension(scenarioID)
			}
			conf, err := unitool.UnmarshalByType(configMap["format"].(string), stream)
			if err == nil {
				configMap["config"] = conf
				if configEntity, err := s.Source.LoadConfigEntity(configMap); err == nil {
					return configEntity, nil
				}
				log.Warnf("LoadConfigEntity error: %v", err)
			} else {
				log.Errorf("UnmarshalByType error: %v", err)
			}
		} else {
			log.Errorf("config map doesn't contain id")
		}
	} else {
		return nil, fmt.Errorf("config entity already loaded: %s", configMap["id"].(string))
	}
	return nil, fmt.Errorf("file %s doesnt'exists", configMap["id"].(string))
}

func (s *SourceEnv) GetIncludeConfigEntityIds(scenarioID string) ([]string, error) {
	ids := make([]string, 0)
	envVars := make([]string, 0)
	if strings.Contains(scenarioID, "_") {
		parts := strings.Split(scenarioID, "_")
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
		ids = append(ids, scenarioID)
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
	return nil, fmt.Errorf("environment variable %s doesnt'exists", configMap["id"].(string))
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
		return nil, fmt.Errorf("config entity already loaded: %s", configMap["id"].(string))
	}
	return nil, fmt.Errorf("source config map entry %s doesnt'exists", configMap["id"].(string))
}

func (s *SourceConfigMap) GetIncludeConfigEntityIds(scenarioID string) ([]string, error) {
	ids := make([]string, 0)
	files := make([]string, 0)
	if strings.Index(scenarioID, "/") == 0 {
		ids = append(ids, scenarioID)
	} else {
		if strings.Contains(scenarioID, "/") {
			s := strings.Split(scenarioID, "/")
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
			ids = append(ids, scenarioID)
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
	if autoloadID, ok := sourceMap["autoload"]; ok {
		source.autoloadID = autoloadID.(string)
	}
	return source
}

func NewSourceGoGetter(sourceName string, url string) *SourceGoGetter {
	sourceMap := map[string]interface{}{
		"name": sourceName,
		"path": path.Join(appTempFilesPath, sourcesStoragePath, sourceName),
	}
	return &SourceGoGetter{
		SourceFile: *NewSourceFile(sourceName, sourceMap),
		url:        url,
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
