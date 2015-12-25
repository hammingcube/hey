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

package build

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const SCRIPT = "local_build.sh"
const TEMPL = "docker run --rm -v {{.Path}}:/app -v {{.Destination}}:/dest -w /app {{.Container}} sh -c"

var ScriptTemplates = map[string]map[string]string{
	"cpp": map[string]string{
		"compile":         "g++ -std=c++11 /app/{{.Src}} -o /dest/{{.Output}}",
		"compile-and-run": "g++ -std=c++11 /app/{{.Src}} -o /dest/exec &&  /dest/exec {{if .Input}} < /app/{{.Input}} {{end}} > /dest/{{.Output}}",
	},
}

var rootDir string

type Config struct {
	Src         string
	Destination string
	Output      string
	Path        string
	Input       string
	Lang        string
	Container   string
	Script      string
	Langauge    string
}

func cwd() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println(dir)
	return dir
}

func MustStr(t string, err error) string {
	if err != nil {
		panic(err)
	}
	return t
}

func dockerCmd(src, outFile string) []string {
	containersMap := map[string]string{
		"cpp":    "glot/clang",
		"golang": "glot/golang",
	}

	lang := "cpp"
	config := &Config{
		Src:         filepath.Base(src),
		Path:        MustStr(filepath.Abs(filepath.Dir(src))),
		Output:      filepath.Base(outFile),
		Destination: MustStr(filepath.Abs(filepath.Dir(outFile))),
		Input:       "", //filepath.Join(filepath.Dir(src), "input.txt"),
		Lang:        lang,
		Container:   containersMap[lang],
	}

	var b bytes.Buffer
	scriptTemplate := template.Must(template.New("script").Parse(ScriptTemplates[lang]["compile-and-run"]))
	err := scriptTemplate.Execute(&b, config)
	if err != nil {
		panic(err)
	}
	config.Script = b.String()

	var b2 bytes.Buffer
	mainTemplate := template.Must(template.New("main").Parse(TEMPL))
	err = mainTemplate.Execute(&b2, config)
	if err != nil {
		panic(err)
	}
	commandSlice := append(strings.Split(b2.String(), " "), fmt.Sprintf("%s", config.Script))
	return commandSlice
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
var BuildCmd = &cobra.Command{
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
		src, outFile := args[0], args[1]
		command := dockerCmd(src, outFile)
		if dryRun {
			//fmt.Printf("%#v\n", command)
			fmt.Printf("%s \"%s\"\n", strings.Join(command[:len(command)-1], " "), command[len(command)-1])
			return
		}
		var out bytes.Buffer
		var stderr bytes.Buffer
		execCmd := exec.Command(command[0], command[1:]...)
		execCmd.Stdout = &out
		execCmd.Stderr = &stderr
		err = execCmd.Run()
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
			return
		}
	},
}

func init() {
	//out := new(bytes.Buffer)
	//fmt.Println(string(out.Bytes()))
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// buildCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	BuildCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Dry run the command")
}
