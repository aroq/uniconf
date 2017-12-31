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

package unitools

import (
	"fmt"
	"github.com/spf13/viper"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"strings"
	"os"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4"
	"github.com/ghodss/yaml"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
)

func ExecCommandString(cmd string) {
	fmt.Println("command is ", cmd)

	parts := strings.Fields(cmd)
	head := parts[0]
	parts = parts[1:]

	out, err := exec.Command(head, parts...).Output()
	if err != nil {
		log.Fatalf("Error: %s", err)
	}
	fmt.Printf("%s\n", out)
}
func ExecCommand(name string, arg ...string) {
	log.Printf("command is %v", name+" "+strings.Join(arg, " "))
	if viper.GetBool("simulate") {
		log.Println("Command executed in simulate mode")
	} else {
		out, err := exec.Command(name, arg...).Output()
		if err != nil {
			log.Fatalf("%s", err)
		}
		fmt.Printf("%s", out)
	}
}

// TODO: Add merge for lists (initial arguments).
func Merge(dst, src map[string]interface{}) map[string]interface{} {
	for k, v := range src {
		if dst == nil {
			dst = make(map[string]interface{})
		}
		if _, ok := dst[k]; !ok { // No key in dst.
			dst[k] = v
		} else {
			switch v.(type) {
			case map[string]interface{}:
				switch dst[k].(type) {
				case map[string]interface{}:
					dst[k] = Merge(dst[k].(map[string]interface{}), v.(map[string]interface{}))
				default:
					dst[k] = v
				}
			case []interface{}:
				switch dst[k].(type) {
				case []interface{}:
					for _, item := range v.([]interface{}) {
						dst[k] = append(dst[k].([]interface{}), item)
					}
				default:
					dst[k] = v
				}
			default:
				dst[k] = v
			}
		}
	}
	return dst
}

func ReadFile(filename string) []byte {
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

		// TODO: Recheck this part.
		//log.Println("Try to clone with 'git' command execution...")
		//unitools.ExecCommandString(fmt.Sprintf("git clone --depth=1 -b %s %s %s", ref, source.Repo, sourcePath))
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

func UnmarshalYaml(yamlFile []byte) (map[string]interface{}, error) {
	y := make(map[string]interface{})
	err := yaml.Unmarshal(yamlFile, &y)
	if err != nil {
		log.Panicf("UnmarshalYaml error: %v", err)
		return nil, err
	}
	return y, nil
}

func UnmarshalEnvVarJson(varName string) (map[string]interface{}, error) {
	envConfigString := os.Getenv(varName)
	if envConfigString != "" {
		var envConfig map[string]interface{}
		json.Unmarshal([]byte(envConfigString), &envConfig)
		return envConfig, nil
	} else {
		// TODO: Provide error message.
		return nil, nil
	}
}


