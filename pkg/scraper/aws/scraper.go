package aws

import (
	"context"

	"sync"

	"github.com/chanzuckerberg/camelot/pkg/scraper/interfaces"
	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
	"github.com/chanzuckerberg/camelot/pkg/util"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// If profile is not passed it is assumed implicitly based on environment variables, like AWS_PROFILE
func Scrape(ctx context.Context, opts ...AWSClientOpt) (*types.InventoryReport, error) {
	awsClient, err := NewAWSClient(ctx, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load config")
	}

	regions := []string{awsClient.GetConfig().Region}
	if awsClient.GetConfig().Region == "" {
		regions = []string{"us-east-1", "us-west-2", "us-east-2", "us-west-1"}
	}

	extractors := []func(ctx context.Context, awsClient interfaces.AWSClient) (*types.InventoryReport, error){
		extractEksClusterInfo,
		extractRds,
		extractLambdas,
		extractAMIs,
		extractVolumes,
		extractACMCertificates,
	}

	var wg sync.WaitGroup
	wg.Add(len(regions) * len(extractors))

	reports := make([]*types.InventoryReport, len(regions)*len(extractors))
	index := 0

	for _, region := range regions {
		logrus.Debugf("Scraping profile %s, region %s", awsClient.GetProfile(), region)
		client, err := NewAWSClient(ctx, append(opts, WithRegion(region))...)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load config for profile %s, region %s", awsClient.GetProfile(), region)
		}

		for _, extractor := range extractors {
			go func(client interfaces.AWSClient, extractor func(ctx context.Context, awsClient interfaces.AWSClient) (*types.InventoryReport, error), i int) {
				defer wg.Done()

				report, err := extractor(ctx, client)
				if err != nil {
					logrus.Errorf("failed to extract inventory: %s", err.Error())
				} else {
					reports[i] = report
				}
			}(client, extractor, index)
			index++
		}
	}
	wg.Wait()

	summary := util.CombineReports(reports)
	summary.Identity = types.Indentity{
		AwsAccountNumber: awsClient.GetAccountId(),
	}

	return &summary, nil
}
