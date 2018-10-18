// Copyright © 2018 Alexander Tolstikov <tolstikov@gmail.com>
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

package cmd

import (
	"fmt"

	"github.com/aroq/uniconf/uniconf"
	"github.com/spf13/cobra"
)

var collectJSONPath string

var collectKey string

// collectCmd represents the collect command
var collectCmd = &cobra.Command{
	Use:   "collect",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(uniconf.Collect(collectJSONPath, collectKey))
	},
}

func init() {
	rootCmd.AddCommand(collectCmd)

	collectCmd.Flags().StringVarP(&collectJSONPath, "jsonpath", "j", ".", "Jsonpath expression to get collect params from ('.' by default)")
	collectCmd.Flags().StringVarP(&collectKey, "key", "k", "params", "Element name to collect params from ('params' by default)")
}
