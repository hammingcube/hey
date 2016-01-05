package build

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

type Options struct {
	Src         string
	OutFile     string
	DryRun      bool
	Language    string
	OnlyCompile bool
}

const TEMPL = "docker run --rm -v {{.Path}}:/app -v {{.Destination}}:/dest -w /app {{.Container}} sh -c"

const RUN_EXEC_WITH_INPUT = "{{if .InputExists}} < /app/{{.Input}} {{end}} > /dest/{{.Output}}"
const CPP_COMPILE = "g++ -std=c++11 /app/{{.Src}} -o /dest/{{.Output}}"
const CPP_COMPILE_AND_RUN = "g++ -std=c++11 /app/{{.Src}} -o /dest/exec && /dest/exec " + RUN_EXEC_WITH_INPUT + " && rm -f /dest/exec"
const PY_COMPILE = "echo '#!/usr/bin/env python' > /dest/{{.Output}} && cat /app/{{.Src}} >> /dest/{{.Output}} && chmod +x /dest/{{.Output}}"
const PY_COMPILE_AND_RUN = "python /app/{{.Src}}" + RUN_EXEC_WITH_INPUT

var containersMap = map[string]string{
	"c":   "gcc:4.9",
	"cpp": "gcc:4.9",
	"go":  "golang",
	"py2": "python:2.7",
	"py3": "python:3",
}

var ScriptTemplates = map[string]map[string]string{
	"c": map[string]string{
		"compile":         CPP_COMPILE,
		"compile-and-run": CPP_COMPILE_AND_RUN,
	},
	"cpp": map[string]string{
		"compile":         CPP_COMPILE,
		"compile-and-run": CPP_COMPILE_AND_RUN,
	},
	"go": map[string]string{
		"compile":         "go build -o /dest/{{.Output}} /app/{{.Src}}",
		"compile-and-run": "go build -o /dest/exec /app/{{.Src}} && /dest/exec && /dest/exec " + RUN_EXEC_WITH_INPUT + " && rm -f /dest/exec",
	},
	"py2": map[string]string{
		"compile":         PY_COMPILE,
		"compile-and-run": PY_COMPILE_AND_RUN,
	},
	"py3": map[string]string{
		"compile":         PY_COMPILE,
		"compile-and-run": PY_COMPILE_AND_RUN,
	},
}

type Config struct {
	Src         string
	Destination string
	Output      string
	Path        string
	Input       string
	InputExists bool
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

func dockerCmd(opts *Options) []string {
	src, lang, outFile := opts.Src, opts.Language, opts.OutFile
	config := &Config{
		Src:         filepath.Base(src),
		Path:        MustStr(filepath.Abs(filepath.Dir(src))),
		Output:      filepath.Base(outFile),
		Destination: MustStr(filepath.Abs(filepath.Dir(outFile))),
		Input:       filepath.Join(filepath.Dir(src), "input.txt"),
		Lang:        lang,
		Container:   containersMap[lang],
	}
	if _, err := os.Stat(config.Input); err == nil {
		config.InputExists = true
	}

	todo := map[bool]string{true: "compile", false: "compile-and-run"}[opts.OnlyCompile]

	var b bytes.Buffer
	scriptTemplate := template.Must(template.New("script").Parse(ScriptTemplates[lang][todo]))
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

func RunFunc(opts *Options) (string, string, error) {
	command := dockerCmd(opts)
	fmt.Printf("%s \"%s\"\n", strings.Join(command[:len(command)-1], " "), command[len(command)-1])
	if opts.DryRun {
		return "", "", nil
	}
	var out bytes.Buffer
	var stderr bytes.Buffer
	execCmd := exec.Command(command[0], command[1:]...)
	execCmd.Stdout = &out
	execCmd.Stderr = &stderr
	err := execCmd.Run()
	return out.String(), stderr.String(), err
}
