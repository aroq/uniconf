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

package cmd

import (
	"os"

	"github.com/aroq/uniconf/uniconf"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"path"
	"fmt"
)

var cfgFile string

var cfgEnvVar string

var outputFormat string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "uniconf",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
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
			},
		})

		uniconf.AddPhase(&uniconf.Phase{
			Name: "process",
			Phases: []*uniconf.Phase{
				{
					Name:     "process",
					Callback: uniconf.ProcessKeys,
					Args: []interface{}{
						"projects",
						"",
						[]*uniconf.Processor{
							{
								Callback:    uniconf.FromProcess,
								IncludeKeys: []string{uniconf.IncludeListElementName},
							},
						},
					},
				},
			},
		})

		uniconf.Execute()
		if outputFormat == "yaml" {
			fmt.Println(uniconf.GetYaml())
		}
		if outputFormat == "json" {
			fmt.Println(uniconf.GetJson())
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global persistent flags.
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config file", "c", path.Join(".unipipe/config.yaml"), "config file ('.unipipe/config.yaml' by default)")
	rootCmd.PersistentFlags().StringVarP(&cfgEnvVar, "config env var", "e", "UNICONF", "config ENV VAR name ('UNICONF' by default)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "yaml", "output format, e.g. 'yaml' or 'json' ('yaml' by default)")
}

// initConfig initializes Uniconf.
func initConfig() {
	uniconf.AddSource(uniconf.NewSourceConfigMap("root", map[string]interface{}{
		"configMap": map[string]interface{}{
			"root": defaultUniconfConfig(),
		},
	}))
	uniconf.SetRootSource("root")
}

// defaultUniconfConfig provides default Uniconf configuration.
func defaultUniconfConfig() map[string]interface{} {
	return map[string]interface{}{
		"sources": map[string]interface{}{
			"env": map[string]interface{}{
				"type": "env",
			},
			"project": map[string]interface{}{
				"type": "file",
				"path": "",
			},
		},
		"from": []interface{}{
			"env:UNICONF",
			"project:/" + cfgFile,
			"env:" + cfgEnvVar,
		},
	}
}
