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
	"strings"
	"log"
	"github.com/spf13/viper"
	"os/exec"
)

func ExecCommandString(cmd string) {
	fmt.Println("command is ", cmd)

	parts := strings.Fields(cmd)
	head := parts[0]
	parts = parts[1:]

	out, err := exec.Command(head, parts...).Output()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}
	fmt.Printf("%s\n", out)
}
func ExecCommand(name string, arg...string) {
	log.Printf("command is %v", name + " " + strings.Join(arg, " "))
	if viper.GetBool("simulate") {
		log.Println("Command executed in simulate mode")
	} else {
		out, err := exec.Command(name, arg...).Output()
		if err != nil {
			fmt.Printf("%s", err)
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
