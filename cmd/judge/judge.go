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

package judge

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/maddyonline/hey/cmd/build"
	"github.com/maddyonline/hey/utils"
	_ "github.com/phayes/hookserve/hookserve"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

func longUsage() string {
	b := &bytes.Buffer{}
	w := new(tabwriter.Writer)
	raw := []string{
		"raw mode:\tSimply run the program and return it's output.",
		"i/o mode:\tJudge with respect to given input/output files.",
		"std mode:\tJudge against a primary-solution using a primary-runner and a primary-generator for test cases."}
	w.Init(b, 0, 8, 0, '\t', 0)
	for _, r := range raw {
		fmt.Fprintln(w, r)
	}
	w.Flush()
	return fmt.Sprintf("Primarily used to judge an input program. Builds and runs an input program and 'judges' the output.\n%s", b)
}

const PROBLEM_CONFIG = "problem-config.json"

var opts = struct {
	DryRun   bool
	Raw      bool
	NoDocker bool
	Language string
}{}

func init() {
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// judgeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	JudgeCmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "d", false, "Dry run the command")
	JudgeCmd.Flags().BoolVarP(&opts.Raw, "raw", "r", false, "Use the raw mode of judging")
	JudgeCmd.Flags().BoolVarP(&opts.NoDocker, "no-docker", "n", false, "Do not use docker")
	JudgeCmd.Flags().StringVarP(&opts.Language, "lang", "l", "", "The programming language of input program")
}

func validateArgs(args []string) (string, string, error) {
	if len(args) < 1 {
		return "", "", errors.New("Need at least one argument (input program directory)")
	}
	if len(args) > 1 {
		return args[0], args[1], nil
	}
	u, err := user.Current()
	if err != nil {
		return "", "", err
	}
	dir, err := utils.CreateDirIfReqd(filepath.Join(u.HomeDir, "hey-judge"))
	if err != nil {
		return "", "", err
	}
	return args[0], filepath.Join(dir, "judge-output.json"), nil
}

func MustStr(v string, err error) string {
	if err != nil {
		panic(err)
	}
	return v
}

func MustBytes(v []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return v
}

// judgeCmd represents the judge command
var JudgeCmd = &cobra.Command{
	Use:   "judge",
	Short: "Builds, runs, and judges an input program",
	Long:  longUsage(),
	Run: func(cmd *cobra.Command, args []string) {
		solnSrc, judgeOutput, err := validateArgs(args)
		if err != nil {
			fmt.Printf("Error validating args: %v\n", err)
			return
		}
		solnDir := MustStr(filepath.Abs(filepath.Dir(solnSrc)))
		judgeDir := MustStr(filepath.Abs(filepath.Dir(judgeOutput)))
		judgeOutputFile := MustStr(filepath.Abs(judgeOutput))

		// Check for at most 4-level of nesting
		possibleConfigFiles := []string{
			filepath.Join(solnDir, PROBLEM_CONFIG),
			filepath.Join(solnDir, "../", PROBLEM_CONFIG),
			filepath.Join(solnDir, "../../", PROBLEM_CONFIG),
			filepath.Join(solnDir, "../../../", PROBLEM_CONFIG),
		}

		probCfg := ""
		for _, probCfg = range possibleConfigFiles {
			if _, err := os.Stat(probCfg); err == nil {
				break
			}
		}
		data, err := ioutil.ReadFile(probCfg)
		//fmt.Println(data)
		//v := map[string]interface{}{}

		type Details struct {
			Url      string `json:"url"`
			Path     string `json:"path"`
			Src      string `json:"src"`
			Lang     string `json:"lang"`
			LocalSrc string `json:"local_src"`
		}

		v := struct {
			PrimarySolution  Details `json:"primary-solution"`
			PrimaryGenerator Details `json:"primary-generator"`
			PrimaryRunner    Details `json:"primary-runner"`
			MySolution       Details `json:"my-solution"`
		}{}
		json.Unmarshal(data, &v)
		var lang string
		if opts.Language != "" {
			lang = opts.Language
		} else {
			parts := strings.Split(solnSrc, ".")
			ext := parts[len(parts)-1]
			lang = map[string]string{"cpp": "cpp", "cc": "cpp", "c": "cpp", "py": "py", "go": "go"}[ext]
		}
		v.MySolution = Details{
			Src:      solnSrc,
			Lang:     lang,
			LocalSrc: MustStr(filepath.Abs(filepath.Join(solnDir, solnSrc))),
		}
		fmt.Printf("%s\n", v)
		rootDir := judgeDir
		workdir := "work_dir"
		//var repo, owner string
		lookFor := filepath.Join(rootDir, workdir, v.PrimarySolution.Url, ".git")
		if _, err := os.Stat(lookFor); err == nil {
			os.Chdir(filepath.Join(rootDir, workdir, v.PrimarySolution.Url))
			out, err := exec.Command("git", "pull").Output()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(out)
		} else {
			dir, _ := filepath.Abs(filepath.Join(rootDir, workdir, v.PrimarySolution.Url))
			err := os.MkdirAll(dir, 0777)
			os.Chdir(filepath.Join(dir, ".."))
			gitUrl := fmt.Sprintf("https://%s", v.PrimarySolution.Url)
			out, err := exec.Command("git", "clone", gitUrl).Output()
			fmt.Printf("%s\n%v\n", out, err)
		}
		setLocalSrc := func(rootDir, workDir string, d *Details) {
			d.LocalSrc = MustStr(filepath.Abs(filepath.Join(rootDir, workDir, d.Url, d.Path, d.Src)))
		}
		setLocalSrc(rootDir, workdir, &v.PrimarySolution)
		setLocalSrc(rootDir, workdir, &v.PrimaryGenerator)
		setLocalSrc(rootDir, workdir, &v.PrimaryRunner)

		fmt.Println("finally\n")
		fmt.Printf("%s\n", MustBytes(json.MarshalIndent(v, "", "    ")))

		buildIt := func(d *Details, outputName string) {
			_, prog_stderr, err := build.RunFunc(&build.Options{
				Src:         d.LocalSrc,
				OutFile:     filepath.Join(rootDir, workdir, outputName),
				DryRun:      false,
				Language:    d.Lang,
				OnlyCompile: true,
			})
			if prog_stderr != "" || err != nil {
				fmt.Printf("Err: %v\nProg Stderr: %s\n", err, prog_stderr)
				panic("Something went wrong")
			}
		}
		buildIt(&v.MySolution, "my-soln")
		buildIt(&v.PrimarySolution, "primary-soln")
		buildIt(&v.PrimaryGenerator, "primary-gen")
		buildIt(&v.PrimaryRunner, "primary-runner")

		destDir := MustStr(filepath.Abs(filepath.Join(rootDir, workdir)))
		runCmd := fmt.Sprintf("docker run --rm -v %s:/app -w /app ubuntu ./primary-runner ./primary-gen ./my-soln ./primary-soln", destDir)
		fmt.Println(runCmd)
		cmds := strings.Split(runCmd, " ")
		_, err = exec.Command(cmds[0], cmds[1:]...).CombinedOutput()
		if err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		readFileAsString := func(filename string) string {
			return string(MustBytes(ioutil.ReadFile(path.Join(destDir, filename))))
		}
		status := readFileAsString("status.json")
		input := readFileAsString("input.txt")
		out1 := readFileAsString("out1.txt")
		out2 := readFileAsString("out2.txt")

		theOutput := struct {
			Status string `json:"status"`
			Input  string `json:"input"`
			Out1   string `json:"out1"`
			Out2   string `json:"out2"`
		}{status, input, out1, out2}
		fmt.Printf("%s\n", MustBytes(json.MarshalIndent(theOutput, "", "    ")))
		fmt.Printf("Writing output to %s\n", judgeOutputFile)
		err = ioutil.WriteFile(judgeOutputFile, MustBytes(json.Marshal(theOutput)), 0755)
		if err != nil {
			panic(err)
		}
	},
}

func sPtr(s string) *string { return &s }

func downloadArchive(url *url.URL, dest string) {
	log.Printf("Downloading: %s\n", url)
	cmd := exec.Command("curl", "-o", dest, "-O", url.String())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}
func extractArchive(zipFile, outputDir string) {
	log.Printf("Extracting...\n")
	cmd := exec.Command("unzip", zipFile, "-d", outputDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func doIt(client *github.Client, owner, repo string, opt *github.RepositoryContentGetOptions) (string, string, string) {
	url, _, err := client.Repositories.GetArchiveLink(owner, repo, github.Zipball, opt)
	if err != nil {
		log.Fatal(err)
	}
	downloadArchive(url, "abc.zip")
	os.Mkdir("unique_dir", 0777)
	extractArchive("abc.zip", "unique_dir")
	dirs, err := ioutil.ReadDir("unique_dir")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Found following directories:\n%v\n", dirs[0].Name())
	filename := path.Join("unique_dir", dirs[0].Name(), "runtests.json")
	data, err := ioutil.ReadFile(filename)
	var runTestsConfig map[string]string
	var problem, mySolnDir string
	json.Unmarshal(data, &runTestsConfig)
	log.Printf("Read:%s\n", runTestsConfig)
	arr := strings.Split(runTestsConfig["runtests"], ",")
	if len(arr) > 1 {
		problem = arr[0]
		mySolnDir = arr[1]
	}
	fmt.Printf("problem: %s, mysolnDir: %s\n", problem, mySolnDir)
	return path.Join("unique_dir", dirs[0].Name()), problem, mySolnDir
}
