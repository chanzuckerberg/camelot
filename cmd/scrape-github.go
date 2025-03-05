package cmd

import (
	"fmt"

	"github.com/chanzuckerberg/camelot/pkg/printer"
	scraper "github.com/chanzuckerberg/camelot/pkg/scraper/github"
	"github.com/chanzuckerberg/camelot/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	flagGithubOrg = "github-org"
)

var (
	scrapeGithubCmd = &cobra.Command{
		Use:   "github",
		Short: "scrapes github for versioned inventory",
		Long:  ``,
		RunE:  scrapeGithub,
	}
	githubOrg string
)

func init() {
	scrapeCmd.AddCommand(scrapeGithubCmd)
	scrapeGithubCmd.Flags().StringVar(&githubOrg, flagGithubOrg, "chanzuckerberg", "Github org to scan. Defaults to chanzuckerberg.")
}

func scrapeGithub(cmd *cobra.Command, args []string) error {
	report, err := scraper.Scrape(cmd.Context(), githubOrg)
	if err != nil {
		return fmt.Errorf("failed to scrape resources: %w", err)
	}
	if report == nil {
		return errors.New("No report was produced")
	}
	logrus.Debug("Scraping complete")

	err = printer.PrintReport(report, util.CreateFilter(filter), outputFormat)
	if err != nil {
		return fmt.Errorf("failed to print report: %w", err)
	}

	return nil
}
