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
	"github.com/aroq/uniconf/unitool"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var contextName string

var contextId string

// contextCmd represents the entity command
var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Set context",
	Long:  `Set context.`,
	Run: func(cmd *cobra.Command, args []string) {
		var context interface{}

		uniconf.AddPhase(&uniconf.Phase{
			Name: "config",
			Phases: []*uniconf.Phase{
				{
					Name:     "load",
					Callback: uniconf.Load,
				},
				{
					Name:     "flatten_config",
					Callback: uniconf.FlattenConfig,
				},
				{
					Name:     "process",
					Callback: uniconf.ProcessKeys,
					Args: []interface{}{
						"jobs",
						"",
						[]*uniconf.Processor{
							{
								Callback:    uniconf.FromProcess,
								IncludeKeys: []string{uniconf.IncludeListElementName},
							},
						},
					},
				},
				{
					Name:     "process_context",
					Callback: uniconf.ProcessContext,
					Args: []interface{}{
						viper.Get("context_name"),
						viper.Get("context_id"),
					},
					Result: &context,
				},
			},
		})

		uniconf.Execute()
		if outputFormat == "yaml" {
			fmt.Println(unitool.MarshallYaml(uniconf.Config()))
		}
		if outputFormat == "json" {
			fmt.Println(unitool.MarshallJson(uniconf.Config()))
		}
	},
}

func init() {
	rootCmd.AddCommand(contextCmd)
	contextCmd.PersistentFlags().StringVarP(&contextName, "name", "n", "", "Context name")
	contextCmd.PersistentFlags().StringVarP(&contextId,   "id", "i", "", "Context id")

	viper.BindPFlag("context_name", contextCmd.PersistentFlags().Lookup("name"))
	viper.BindPFlag("context_id",   contextCmd.PersistentFlags().Lookup("id"))

	viper.AutomaticEnv()
	viper.SetEnvPrefix("UNICONF")
}
