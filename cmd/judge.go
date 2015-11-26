// Copyright © 2015 NAME HERE <EMAIL ADDRESS>
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
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path/filepath"
)

const PROBLEM_CONFIG = "problem-config.json"

// judgeCmd represents the judge command
var judgeCmd = &cobra.Command{
	Use:   "judge",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		solnDir := "."
		if len(args) > 0 {
			solnDir = args[0]
		}
		solnDir, err := filepath.Abs(solnDir)
		if err != nil {
			fmt.Println("Error resolving path: %s", err)
			return
		}
		// Check for at most 4-level of nesting
		possibleConfigFiles := []string{
			filepath.Join(solnDir, PROBLEM_CONFIG),
			filepath.Join(solnDir, "../", PROBLEM_CONFIG),
			filepath.Join(solnDir, "../../", PROBLEM_CONFIG),
			filepath.Join(solnDir, "../../../", PROBLEM_CONFIG),
		}
		fmt.Println(possibleConfigFiles)
		probCfg := ""
		for _, probCfg = range possibleConfigFiles {
			fmt.Println(probCfg)
			if _, err := os.Stat(probCfg); err == nil {
				break
			}
		}
		fmt.Println(probCfg)
		data, err := ioutil.ReadFile(probCfg)
		//fmt.Println(data)
		//v := map[string]interface{}{}

		type Location struct {
			Url  string `json:"url"`
			Path string `json:"path"`
		}

		v := struct {
			PrimarySolution  Location `json:"primary-solution"`
			PrimaryGenerator Location `json:"primary-generator"`
			PrimaryRunner    Location `json:"primary-runner"`
		}{}

		json.Unmarshal(data, &v)
		fmt.Println(v)
	},
}

func init() {
	RootCmd.AddCommand(judgeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// judgeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// judgeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
