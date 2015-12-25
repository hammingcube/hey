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
	. "github.com/maddyonline/hey/cmd/common"
	_ "github.com/phayes/hookserve/hookserve"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
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
	DryRun         bool
	Raw            bool
	DoNotUseDocker bool
	Language       string
}{}

func init() {
	RootCmd.AddCommand(judgeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// judgeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	judgeCmd.Flags().BoolVarP(&opts.DryRun, "dry-run", "d", false, "Dry run the command")
	judgeCmd.Flags().BoolVarP(&opts.Raw, "raw", "r", false, "Use the raw mode of judging")
	judgeCmd.Flags().BoolVarP(&opts.DoNotUseDocker, "without-docker", "w", false, "Do not use docker")
	judgeCmd.Flags().StringVarP(&opts.Langauge, "language", "l", "", "The programming language of input program")
}

func validateArgs(args []string) error {
	if len(args) < 1 {
		return errors.New("Need at least one argument (input program directory)")
	}
	return nil
}

// judgeCmd represents the judge command
var judgeCmd = &cobra.Command{
	Use:   "judge",
	Short: "Builds, runs, and judges an input program",
	Long:  longUsage(),
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateArgs(args); err != nil {
			fmt.Printf("%v\n", err)
			return
		}
		solnDir = args[0]

		fmt.Printf("%v\n", opts)
		return
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
		rootDir, _ := filepath.Abs(".")
		workdir := "work_dir"
		//var repo, owner string
		fmt.Println(v.PrimarySolution.Url)
		lookFor := filepath.Join(workdir, v.PrimarySolution.Url, ".git")
		fmt.Println(lookFor)

		if _, err := os.Stat(lookFor); err == nil {
			fmt.Println("Found Directory")
			os.Chdir(filepath.Join(workdir, v.PrimarySolution.Url))
			out, err := exec.Command("git", "pull").Output()
			if err != nil {
				log.Fatal(err)
			}
			fmt.Printf("The output of command is %s\n", out)
		} else {
			fmt.Println("Cannot find directory")
			dir, _ := filepath.Abs(filepath.Join(workdir, v.PrimarySolution.Url))
			fmt.Printf("Making %s directory\n", dir)
			err := os.MkdirAll(dir, 0777)
			os.Chdir(filepath.Join(dir, ".."))
			gitUrl := fmt.Sprintf("https://%s", v.PrimarySolution.Url)
			fmt.Println(gitUrl)
			out, err := exec.Command("git", "clone", gitUrl).Output()
			fmt.Printf("out: %s, err: %s\n", out, err)
		}
		fmt.Println(rootDir)
		primary_soln := filepath.Join(rootDir, workdir, v.PrimarySolution.Url, v.PrimarySolution.Path)
		gen := filepath.Join(rootDir, workdir, v.PrimaryGenerator.Url, v.PrimaryGenerator.Path)
		runtest := filepath.Join(rootDir, workdir, v.PrimaryRunner.Url, v.PrimaryRunner.Path)

		buildCmd.Run(nil, []string{runtest, "runtest"})
		buildCmd.Run(nil, []string{gen, "gen"})
		buildCmd.Run(nil, []string{solnDir, "my-soln"})
		buildCmd.Run(nil, []string{primary_soln, "primary-soln"})

		destDir := filepath.Join(rootDirectory, destDirectory)
		runCmd := fmt.Sprintf("docker run --rm -v %s:/app -w /app ubuntu ./runtest ./gen ./my-soln ./primary-soln", destDir)
		cmds := strings.Split(runCmd, " ")
		finalOutput, err := exec.Command("docker", cmds[1:]...).CombinedOutput()
		fmt.Println(runCmd)
		fmt.Println(finalOutput)
		fmt.Println(err)
		jsonBytes, err := ioutil.ReadFile(path.Join(destDir, "status.json"))
		fmt.Println(string(jsonBytes))
		fmt.Println()

		//c5 := fmt.Sprintf(runCmd, binDir)

		//client := github.NewClient(nil)
		//opt := &github.RepositoryContentGetOptions{"master"}
		//doIt(client, "maddyonline", "epibook.github.io", opt)

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
