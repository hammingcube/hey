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
	"bytes"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	_ "os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

const SCRIPT = "local_build.sh"
const TEMPL = "docker run --rm -v {{.Path}}:/app -v {{.Destination}}:/dest -w /app {{.Container}} sh {{.Script}} /dest/{{.Output}}"

type Config struct {
	Path        string
	Container   string
	Script      string
	Destination string
	Output      string
}

func cwd() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println(dir)
	return dir
}

func dockerCmd(scriptPath, outFile, destDir string) string {
	//fmt.Println(scriptPath)
	destDir = path.Join(cwd(), destDir)
	fullPath, err := filepath.Abs(scriptPath)
	tmpl, err := template.New("test").Parse(TEMPL)
	if err != nil {
		panic(err)
	}
	containersMap := map[string]string{
		"cpp":    "glot/clang",
		"golang": "glot/golang",
	}

	scriptSrc := path.Join(scriptPath, SCRIPT)
	script, err := ioutil.ReadFile(scriptSrc)
	lines := strings.Split(string(script), "\n")
	var lang string
	//fmt.Println(lines[0])
	fmt.Sscanf(lines[0], "# Language: %s", &lang)
	//fmt.Println(lang)

	config := &Config{
		Path:        fullPath,
		Container:   containersMap[lang],
		Script:      SCRIPT,
		Destination: destDir,
		Output:      outFile,
	}

	var b bytes.Buffer
	err = tmpl.Execute(&b, config)
	if err != nil {
		panic(err)
	}
	return b.String()
}

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		destDir := "work_dir"
		for _, arg := range args {
			c1 := dockerCmd(arg, "primary-soln", destDir)
			fmt.Println(c1)
		}
	},
}

func init() {
	RootCmd.AddCommand(buildCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// buildCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// buildCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
