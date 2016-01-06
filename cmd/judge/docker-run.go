package judge

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/maddyonline/hey/cmd/build"
	_ "github.com/phayes/hookserve/hookserve"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type FinalOutput struct {
	Status string `json:"status"`
	Input  string `json:"input"`
	Out1   string `json:"out1"`
	Out2   string `json:"out2"`
}

func RunFunc(data []byte, opts *Options, solnSrc, judgeOutput string) (*FinalOutput, error) {
	judgeDir := MustStr(filepath.Abs(filepath.Dir(judgeOutput)))
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
		LocalSrc: MustStr(filepath.Abs(solnSrc)),
	}
	log.Info("%#v", v)
	rootDir := judgeDir
	workdir := "work_dir"
	//var repo, owner string
	lookFor := filepath.Join(rootDir, workdir, v.PrimarySolution.Url, ".git")
	if _, err := os.Stat(lookFor); err == nil {
		os.Chdir(filepath.Join(rootDir, workdir, v.PrimarySolution.Url))
		out, err := exec.Command("git", "pull").Output()
		if err != nil {
			log.Info("git pull: %q", string(out))
			return &FinalOutput{}, err
		}
	} else {
		dir, _ := filepath.Abs(filepath.Join(rootDir, workdir, v.PrimarySolution.Url))
		err := os.MkdirAll(dir, 0777)
		os.Chdir(filepath.Join(dir, ".."))
		gitUrl := fmt.Sprintf("https://%s", v.PrimarySolution.Url)
		out, err := exec.Command("git", "clone", gitUrl).Output()
		if err != nil {
			log.Info("git clone: %s\n%v\n", out, err)
			return &FinalOutput{}, err
		}
	}
	setLocalSrc := func(rootDir, workDir string, d *Details) {
		d.LocalSrc = MustStr(filepath.Abs(filepath.Join(rootDir, workDir, d.Url, d.Path, d.Src)))
	}
	setLocalSrc(rootDir, workdir, &v.PrimarySolution)
	setLocalSrc(rootDir, workdir, &v.PrimaryGenerator)
	setLocalSrc(rootDir, workdir, &v.PrimaryRunner)

	log.Info("judge-docker-run:")
	log.Info("%q", MustBytes(json.MarshalIndent(v, "", "    ")))

	buildIt := func(d *Details, outputName string) error {
		_, prog_stderr, err := build.RunFunc(&build.Options{
			Src:         d.LocalSrc,
			OutFile:     filepath.Join(rootDir, workdir, outputName),
			DryRun:      false,
			Language:    d.Lang,
			OnlyCompile: true,
		})
		if prog_stderr != "" || err != nil {
			fmt.Printf("Err: %v\nProg Stderr: %s\n", err, prog_stderr)
			err := errors.New(fmt.Sprintf("err: %v, errors: %s", err, prog_stderr))
			return err
		} else {
			return nil
		}
	}
	err1 := buildIt(&v.MySolution, "my-soln")
	err2 := buildIt(&v.PrimarySolution, "primary-soln")
	err3 := buildIt(&v.PrimaryGenerator, "primary-gen")
	err4 := buildIt(&v.PrimaryRunner, "primary-runner")
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return &FinalOutput{}, errors.New(fmt.Sprintf("%v, %v, %v, %v", err1, err2, err3, err4))
	}

	destDir := MustStr(filepath.Abs(filepath.Join(rootDir, workdir)))
	runCmd := fmt.Sprintf("docker run --rm -v %s:/app -w /app ubuntu ./primary-runner ./primary-gen ./my-soln ./primary-soln", destDir)
	fmt.Println(runCmd)
	cmds := strings.Split(runCmd, " ")
	cmdOutput, err := exec.Command(cmds[0], cmds[1:]...).CombinedOutput()
	log.Info("cmdOutput: %s", cmdOutput)
	if err != nil {
		log.Info("%v", err)
		return &FinalOutput{}, err
	}
	readFileAsString := func(filename string) string {
		return string(MustBytes(ioutil.ReadFile(path.Join(destDir, filename))))
	}
	status := readFileAsString("status.json")
	input := readFileAsString("input.txt")
	out1 := readFileAsString("out1.txt")
	out2 := readFileAsString("out2.txt")

	theOutput := &FinalOutput{status, input, out1, out2}
	log.Info("FinalOutput: %#v", theOutput)
	return theOutput, nil
}
