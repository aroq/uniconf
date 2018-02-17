// Copyright © 2017 Alexander Tolstikov <tolstikov@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package unitool

import (
	"github.com/spf13/cast"
	log "github.com/sirupsen/logrus"
	"strings"
	"os"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4"
	"github.com/ghodss/yaml"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"bytes"
	"encoding/gob"
	"errors"
)

var MergeHistory map[string][]string

func init() {
	gob.Register(map[string]interface{}{})
	gob.Register(map[string]string{})
	gob.Register(map[string][]string{})
	gob.Register([]interface{}{})
	MergeHistory = make(map[string][]string)
}

// TODO: Add merge for lists (initial arguments).
func Merge(dst, src interface{}) interface{} { return merge(dst, src, false, "", "") }
func merge(dst, src interface{}, withHistory bool, id string, path string) interface{} {
	if src == nil {
		return dst
	}
	switch src.(type) {
	case map[string]interface{}:
		for k, v := range src.(map[string]interface{}) {
			if dst == nil {
				dst = make(map[string]interface{})
			}
			if _, ok := dst.(map[string]interface{})[k]; !ok { // No key in dst.
				dst.(map[string]interface{})[k] = v
			} else {
				switch v.(type) {
				case map[string]interface{}:
					switch dst.(map[string]interface{})[k].(type) {
					case map[string]interface{}:
						dst.(map[string]interface{})[k] = merge(dst.(map[string]interface{})[k].(map[string]interface{}), v.(map[string]interface{}), withHistory, id + "_" + k, "." + k)
					default:
						dst.(map[string]interface{})[k] = v
					}
				case []interface{}:
					switch dst.(map[string]interface{})[k].(type) {
					case []interface{}:
						for _, item := range v.([]interface{}) {
							dst.(map[string]interface{})[k] = append(dst.(map[string]interface{})[k].([]interface{}), item)
						}
					default:
						dst.(map[string]interface{})[k] = v
					}
				case []string:
					switch dst.(map[string]interface{})[k].(type) {
					case []string:
						for _, item := range v.([]string) {
							if !StringListContains(dst.(map[string]interface{})[k].([]string), item) {
								dst.(map[string]interface{})[k] = append(dst.(map[string]interface{})[k].([]string), item)
							}
						}
					default:
						dst.(map[string]interface{})[k] = v
					}
				default:
					dst.(map[string]interface{})[k] = v
				}
			}
		}
	}
	return dst
}

func ReadFile(filename string) []byte {
	log.Debugf("Read file: %s", filename)
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Panicf("ReadFile error: %v ", err)
	}
	return f
}

func GitClone(url, referenceName, path string, depth int, singleBranch bool) (error) {
	log.Printf("Clone repo: %s", url)
	oldStdout, oldStderr := disableStdStreams(true, false)
	_, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:           url,
		Progress:      os.Stdout,
		SingleBranch:  singleBranch,
		Depth:         depth,
		ReferenceName: plumbing.ReferenceName(referenceName),
	})
	enableStdStreams(oldStdout, oldStderr)
	if err != nil {
		log.Printf("Error: %s", err)
		return err
	} else {
		return nil
	}
}

func disableStdStreams(disableStdout, disableStderr bool) (oldStdout, oldStderr *os.File) {
	if disableStdout {
		oldStdout = os.Stdout
		stdoutFile := filepath.Join(os.TempDir(), "unitools_stdout")
		temp, _ := os.Create(stdoutFile)
		os.Stdout = temp
	}
	if disableStderr {
		oldStderr = os.Stderr
		stderrFile := filepath.Join(os.TempDir(), "unitools_stderr")
		temp2, _ := os.Create(stderrFile)
		os.Stderr = temp2
	}
	return
}

func enableStdStreams(oldStdout, oldStderr *os.File) {
	// Restore all stdout & stderr output.
	os.Stdout = oldStdout
	os.Stderr = oldStderr
}


func UnmarshalByType(t string, stream []byte) (map[string]interface{}, error) {
	if t == "yaml" {
		return UnmarshalYaml(stream)
	}
	if t == "json" {
		return UnmarshalJson(stream)
	}
	return nil, errors.New("unknown type")
}

func UnmarshalYaml(stream []byte) (map[string]interface{}, error) {
	y := make(map[string]interface{})
	err := yaml.Unmarshal(stream, &y)
	if err != nil {
		log.Fatalf("UnmarshalYaml error: %v", err)
		return nil, err
	}
	return y, nil
}

func UnmarshalJson(stream []byte) (map[string]interface{}, error) {
	y := make(map[string]interface{})
	err := json.Unmarshal(stream, &y)
	if err != nil {
		log.Panicf("UnmarshalYaml error: %v", err)
		return nil, err
	}
	return y, nil
}

func MarshallYaml(m interface{}) string {
	y, err := yaml.Marshal(m)
	if err != nil {
		log.Fatalf("Err: %v", err)
	}
	return "---\n" + string(y)
}

func MarshallJson(m interface{}) string {
	y, err := json.Marshal(m)
	if err != nil {
		log.Fatalf("Err: %v", err)
	}
	return string(y)
}

func SearchMapWithPathStringPrefixes(source map[string]interface{}, path string) interface{} {
	path = strings.Trim(path,".")
	return SearchMapWithPathPrefixes(source, strings.Split(path, "."))
}

// searchMapWithPathPrefixes recursively searches for a value for path in source map.
//
// Taken from "viper".
// While searchMap() considers each path element as a single map key, this
// function searches for, and prioritizes, merged path elements.
// e.g., if in the source, "foo" is defined with a sub-key "bar", and "foo.bar"
// is also defined, this latter value is returned for path ["foo", "bar"].
//
// This should be useful only at config level (other maps may not contain dots
// in their keys).
//
// Note: This assumes that the path entries and map keys are lower cased.
func SearchMapWithPathPrefixes(source map[string]interface{}, path []string) interface{} {
	if len(path) == 0 {
		return source
	}

	// search for path prefixes, starting from the longest one
	for i := len(path); i > 0; i-- {
		prefixKey := strings.Join(path[0:i], ".")

		next, ok := source[prefixKey]
		if ok {
			// Fast path
			if i == len(path) {
				return next
			}

			// Nested case
			var val interface{}
			switch next.(type) {
			case map[interface{}]interface{}:
				val = SearchMapWithPathPrefixes(cast.ToStringMap(next), path[i:])
			case map[string]interface{}:
				// Type assertion is safe here since it is only reached
				// if the type of `next` is the same as the type being asserted
				val = SearchMapWithPathPrefixes(next.(map[string]interface{}), path[i:])
			default:
				// got a value but nested key expected, do nothing and look for next prefix
			}
			if val != nil {
				return val
			}
		}
	}

	// not found
	return nil
}

func DeepCollectParams(source map[string]interface{}, path, key string) (map[string]interface{}, error) {
	source, err := DeepCopyMap(source)
	if err != nil {
		return nil, err
	}
	path = strings.Trim(path,".")
	pathParts := strings.Split(path, ".")
	params := make(map[string]interface{})
	p := ""
	for i := 0; i < len(pathParts); i++ {
		if p != "" {
			p += "." + pathParts[i]
		} else {
			p += pathParts[i]
		}
		result := SearchMapWithPathStringPrefixes(source, p + "." + key)
		if result != nil {
			params = Merge(params, result).(map[string]interface{})
		}
	}
	params, err = DeepCopyMap(params)
	return params, err
}

// DeepCollectChildren collects params from nesting structures
// For example: jobs.dev.jobs.install - to collect params from this structure pass path=dev.install and key=jobs.
func DeepCollectChildren(source map[string]interface{}, path, key string) (map[string]interface{}, error) {
	source, err := DeepCopyMap(source)
	if err != nil {
		return nil, err
	}
	path = strings.Trim(path,".")
	pathParts := strings.Split(path, ".")
	params := make(map[string]interface{})
	p := ""
	for i := 0; i < len(pathParts); i++ {
		if p != "" {
			p += "." + key + "." + pathParts[i]
		} else {
			p += key + "." + pathParts[i]
		}
		result := SearchMapWithPathStringPrefixes(source, p)
		if result != nil {
			result, _ := DeepCopyMap(result.(map[string]interface{}))
			delete(result, key)
			params = Merge(params, result).(map[string]interface{})
		}
	}
	return params, nil
}

// DeepCopyMap performs a deep copy of the given map m.
func DeepCopyMap(m map[string]interface{}) (map[string]interface{}, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	dec := gob.NewDecoder(&buf)
	err := enc.Encode(m)
	if err != nil {
		return nil, err
	}
	var copy map[string]interface{}
	err = dec.Decode(&copy)
	if err != nil {
		return nil, err
	}
	return copy, nil
}

func StringListContains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func FormatByExtension(f string) string {
	extension := filepath.Ext(f)
	switch extension {
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	}
	return ""
}
