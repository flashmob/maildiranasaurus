/*             _
              / _)
     _.----._/ /
    /         /
 __/ (  | (  |
/__.-'|_|--|_|

(art credits unknown)

 */
package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "maildiranasaurus",
	Short: "SMTP server using maildir",
	Long:  `It's a small SMTP server uisng go-guerrilla as a package to save email to maildir.`,
	Run:   nil,
}

var (
	verbose bool
)

func init() {
	cobra.OnInitialize()
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false,
		"print out more debug information")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if verbose {
			logrus.SetLevel(logrus.DebugLevel)
		} else {
			logrus.SetLevel(logrus.InfoLevel)
		}
	}
}
