package scraper

import (
	"github.com/chanzuckerberg/camelot/pkg/scraper/aws"
	"github.com/chanzuckerberg/camelot/pkg/scraper/github"
	"github.com/chanzuckerberg/camelot/pkg/scraper/tfc"
	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
)

func ScrapeAWS(profile string) (*types.InventoryReport, error) {
	return aws.Scrape(profile)
}

func ScrapeGithub(githubOrg string) (*types.InventoryReport, error) {
	return github.Scrape(githubOrg)
}

func ScrapeTFC(orgName string) (*types.InventoryReport, error) {
	return tfc.Scrape()
}
