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
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

const SCRIPT = "local_build.sh"

var rootDir string

func validateArgs(args []string) error {
	if len(args) < 2 {
		return errors.New("Need at least two arguments (source and binary)")
	}
	return nil
}

var (
	dryRun      bool
	onlyCompile bool
)

func init() {
	BuildCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Dry run the command")
	BuildCmd.Flags().BoolVarP(&onlyCompile, "only-compile", "c", false, "Only compile/build the program")
}

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
		runFunc(src, outFile, dryRun)
	},
}
