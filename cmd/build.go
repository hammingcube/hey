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
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

const SCRIPT = "local_build.sh"
const TEMPL = "docker run --rm -v {{.Path}}:/app -v {{.Destination}}:/dest -w /app {{.Container}} sh {{.Script}} /dest/{{.Output}}"

var rootDir string

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
	destDir = destDir
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

func validateArgs(args []string) error {
	if len(args) < 2 {
		return errors.New("Need at least two arguments (source and binary)")
	}
	return nil
}

var (
	dryRun bool
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: fmt.Sprintf("Builds a program based on %s", SCRIPT),
	Long: fmt.Sprintf(`Builds a program stored on local filesystem. The input is the path to directory. 
The input directory must contain file %s which is used to build the file.`, SCRIPT),
	Run: func(cmd *cobra.Command, args []string) {
		err := validateArgs(args)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}
		src, binary := args[0], args[1]
		destDir := filepath.Join(rootDirectory, destDirectory)
		destDir, err = createDirIfReqd(destDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return
		}
		command := dockerCmd(src, binary, destDir)
		if dryRun {
			fmt.Println(command)
			return
		}
		dockerCmd := strings.Split(command, " ")
		out, err := exec.Command(dockerCmd[0], dockerCmd[1:]...).Output()
		fmt.Printf("out: %v, err: %v\n", out, err)
	},
}

func init() {
	RootCmd.AddCommand(buildCmd)
	//out := new(bytes.Buffer)
	//fmt.Println(string(out.Bytes()))
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// buildCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	buildCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Dry run the command")

}
