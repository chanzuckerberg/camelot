package scraper

import (
	"context"

	"github.com/chanzuckerberg/camelot/pkg/scraper/aws"
	"github.com/chanzuckerberg/camelot/pkg/scraper/github"
	"github.com/chanzuckerberg/camelot/pkg/scraper/tfc"
	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
)

// Depends on env.AWS_PROFILE
func ScrapeAWS(ctx context.Context, opts ...aws.AWSClientOpt) (*types.InventoryReport, error) {
	return aws.Scrape(ctx, opts...)
}

// Depends on env.GITHUB_TOKEN
func ScrapeGithub(ctx context.Context, githubOrg string) (*types.InventoryReport, error) {
	return github.Scrape(ctx, githubOrg)
}

// Depends on env.TFE_TOKEN and env.TFE_ADDRESS
func ScrapeTFC(ctx context.Context, orgName string) (*types.InventoryReport, error) {
	return tfc.Scrape(ctx)
}
