package cmd

import (
	"github.com/chanzuckerberg/camelot/pkg/printer"
	scraper "github.com/chanzuckerberg/camelot/pkg/scraper/tfc"
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

func scrapeTfc(ccmd *cobra.Command, args []string) error {
	report, err := scraper.Scrape()
	if err != nil {
		return errors.Wrap(err, "failed to scrape resources")
	}
	if report == nil {
		return errors.New("No report was produced")
	}
	logrus.Debug("Scraping complete")

	err = printer.PrintReport(report, outputFormat)
	if err != nil {
		return errors.Wrap(err, "failed to print report")
	}

	return nil
}
