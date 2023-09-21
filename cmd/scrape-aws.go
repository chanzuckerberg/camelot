package cmd

import (
	"github.com/chanzuckerberg/camelot/pkg/printer"
	scraper "github.com/chanzuckerberg/camelot/pkg/scraper/aws"
	"github.com/chanzuckerberg/camelot/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	flagAll = "all"
)

var (
	scrapeAwsCmd = &cobra.Command{
		Use:   "aws",
		Short: "scrapes AWS for versioned inventory",
		Long:  ``,
		RunE:  scrape,
	}
	scanAll bool
)

func init() {
	scrapeCmd.AddCommand(scrapeAwsCmd)
	scrapeAwsCmd.Flags().BoolVarP(&scanAll, flagAll, "a", false, "Scan all aws profiles")
}

func scrape(cmd *cobra.Command, args []string) error {
	var err error
	profiles := []string{""}
	accountMap := map[string]bool{}

	if scanAll {
		profiles, err = scraper.GetAWSProfiles()
		if err != nil {
			return errors.Wrap(err, "failed to get AWS profiles")
		}
	}

	for _, profile := range profiles {
		awsClient, err := scraper.NewAWSClient(cmd.Context(), scraper.WithProfile(profile))
		if err != nil {
			logrus.Errorf("failed to load config for profile %s: %s", profile, err.Error())
			continue
		}
		accountNumber := awsClient.GetAccountId()
		if err != nil {
			logrus.Debugf("failed to get account number for profile %s: %s", profile, err.Error())
			continue
		}

		if _, ok := accountMap[accountNumber]; ok {
			continue
		}
		accountMap[accountNumber] = true

		report, err := scraper.Scrape(cmd.Context(), scraper.WithProfile(profile))
		if err != nil {
			logrus.Errorf("failed to scrape resources for profile %s: %s", profile, err.Error())
		}

		err = printer.PrintReport(report, util.CreateFilter(filter), outputFormat)
		if err != nil {
			logrus.Errorf("failed to print report for profile %s: %s", profile, err.Error())
		}
	}

	logrus.Debug("Scraping complete")
	return nil
}
