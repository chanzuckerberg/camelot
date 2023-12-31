package cmd

import (
	"fmt"

	"github.com/chanzuckerberg/camelot/pkg/util"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of camelot",
	RunE: func(cmd *cobra.Command, args []string) error {
		v, e := util.VersionString()
		if e != nil {
			return e
		}
		fmt.Println(v)
		return nil
	},
}
