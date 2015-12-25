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

const TEMPL = "docker run --rm -v {{.Path}}:/app -v {{.Destination}}:/dest -w /app {{.Container}} sh -c"

var ScriptTemplates = map[string]map[string]string{
	"cpp": map[string]string{
		"compile":         "g++ -std=c++11 /app/{{.Src}} -o /dest/{{.Output}}",
		"compile-and-run": "g++ -std=c++11 /app/{{.Src}} -o /dest/exec &&  /dest/exec {{if .InputExists}} < /app/{{.Input}} {{end}} > /dest/{{.Output}}",
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

func dockerCmd(src, outFile string, onlyCompile bool) []string {
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
		Input:       filepath.Join(filepath.Dir(src), "input.txt"),
		Lang:        lang,
		Container:   containersMap[lang],
	}
	if _, err := os.Stat(config.Input); err == nil {
		config.InputExists = true
	}

	todo := map[bool]string{true: "compile", false: "compile-and-run"}[onlyCompile]

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

func runFunc(src, outFile string, dryRun bool) {
	command := dockerCmd(src, outFile, onlyCompile)
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
	err := execCmd.Run()
	if err != nil {
		fmt.Println(fmt.Sprint(err) + stderr.String())
		return
	}
}