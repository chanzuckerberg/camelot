package cmd

import (
	"fmt"

	"github.com/chanzuckerberg/camelot/pkg/printer"
	scraper "github.com/chanzuckerberg/camelot/pkg/scraper/tfc"
	"github.com/chanzuckerberg/camelot/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	scrapeTfcCmd = &cobra.Command{
		Use:   "tfc",
		Short: "scrapes tfc/tfe for inventory across all organizations",
		Long:  ``,
		RunE:  scrapeTfc,
	}
)

func init() {
	scrapeCmd.AddCommand(scrapeTfcCmd)
}

func scrapeTfc(cmd *cobra.Command, args []string) error {
	report, err := scraper.Scrape(cmd.Context())
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
