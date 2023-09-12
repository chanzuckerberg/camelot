package scraper

import (
	"github.com/chanzuckerberg/camelot/pkg/scraper/aws"
	"github.com/chanzuckerberg/camelot/pkg/scraper/github"
	"github.com/chanzuckerberg/camelot/pkg/scraper/tfc"
	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
)

// Depends on env.AWS_PROFILE
func ScrapeAWS(profile string) (*types.InventoryReport, error) {
	return aws.Scrape(profile)
}

// Depends on env.GITHUB_TOKEN
func ScrapeGithub(githubOrg string) (*types.InventoryReport, error) {
	return github.Scrape(githubOrg)
}

// Depends on env.TFE_TOKEN and env.TFE_ADDRESS
func ScrapeTFC(orgName string) (*types.InventoryReport, error) {
	return tfc.Scrape()
}
