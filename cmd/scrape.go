package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	flagOutputFormat = "output"
	flagFilter       = "filter"
)

var (
	scrapeCmd = &cobra.Command{
		Use:   "scrape",
		Short: "scrapes AWS for versioned inventory",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			logrus.Info("Please specify a subcommand. See --help for more information.")
			return nil
		},
	}
	outputFormat string
	filter       string
)

func init() {
	rootCmd.AddCommand(scrapeCmd)
	scrapeCmd.PersistentFlags().StringVarP(&outputFormat, flagOutputFormat, "o", "text", "Output format (json, yaml, text). Defaults to text.")
	scrapeCmd.PersistentFlags().StringVarP(&outputFormat, flagFilter, "f", "", "Report filter. Defaults to empty.")
}
