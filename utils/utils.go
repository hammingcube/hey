package utils

import (
	"github.com/labstack/gommon/log"
	"io/ioutil"
	"os"
	"path/filepath"
)

func UpdateFile(file, val string) error {
	dir := filepath.Dir(file)
	log.Info("creating dir %s", dir)
	dir, err := CreateDirIfReqd(dir)
	if err != nil {
		return err
	}
	ioutil.WriteFile(filepath.Join(dir, filepath.Base(file)), []byte(val), 0777)
	return nil
}

func CreateDirIfReqd(dir string) (string, error) {
	dirAbsPath, err := filepath.Abs(dir)
	if err != nil {
		return dirAbsPath, err
	}
	if _, err := os.Stat(dirAbsPath); err == nil {
		return dirAbsPath, nil
	}
	err = os.MkdirAll(dirAbsPath, 0777)
	return dirAbsPath, err
}

// if _, err := os.Stat(lookFor); err == nil {
// 	fmt.Println("Found Directory")
// 	os.Chdir(filepath.Join(workdir, v.PrimarySolution.Url))
// 	out, err := exec.Command("git", "pull").Output()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	fmt.Printf("The output of command is %s\n", out)
// } else {
// 	fmt.Println("Cannot find directory")
// 	dir, _ := filepath.Abs(filepath.Join(workdir, v.PrimarySolution.Url))
// 	fmt.Printf("Making %s directory\n", dir)
// 	err := os.MkdirAll(dir, 0777)
