// Copyright © 2018 Lee Briggs <lee@leebriggs.co.uk>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/hashicorp/go-getter"
	"github.com/spf13/cobra"
)

var (
	url string
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize kr8 config repos, components and clusters",
	Long: `kr8 requires specific directories and exists for its config to work.
This init command helps in creating directory structure for repos, clusters and 
components`,
	//Run: func(cmd *cobra.Command, args []string) {},
}

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Initialize a new kr8 config repo",
	Long: `Initialize a new kr8 config repo by downloading the kr8 config skeletion repo
and initialize a git repo so you can get started`,
	Run: func(cmd *cobra.Command, args []string) {

		// Get the current working directory
		pwd, err := os.Getwd()
		if err != nil {
			log.Fatal("Error getting working directory:", err)
		}

		// Download the skeletion directory
		log.Debug("Downloading skeleton repo from ", url)
		client := &getter.Client{
			Src:  url,
			Dst:  base,
			Pwd:  pwd,
			Mode: getter.ClientModeAny,
		}

		if err := client.Get(); err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
	},
}

func init() {
	RootCmd.AddCommand(initCmd)
	initCmd.AddCommand(repoCmd)

	repoCmd.PersistentFlags().StringVar(&url, "url", "git::https://github.com/apptio/kr8-config-skel", "Source of skeleton directory to create repo from")

}
