package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/chanzuckerberg/camelot/pkg/scraper/interfaces"
	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func extractRds(ctx context.Context, awsClient interfaces.AWSClient) (*types.InventoryReport, error) {
	cycleMap := map[string]types.ProductCycle{}
	currentCycleMap := map[string]string{}
	products := []string{"amazon-rds-postgresql", "amazon-rds-mysql"}
	productPrefixes := []string{"aurora-postgresql", "aurora-mysql"}

	for i, product := range products {
		cycles, err := endOfLife(product)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to get %s end of life data", product)
		}

		for index, cycle := range *cycles {
			if index == 0 {
				currentCycleMap[productPrefixes[i]] = (*cycles)[0].Cycle
			}
			cycleMap[productPrefixes[i]+"-"+cycle.Cycle] = cycle
		}
	}

	rdsClusters := []types.RDSCluster{}

	out, err := awsClient.DescribeRDSClusters()
	if err != nil {
		logrus.Errorf("unable to list clusters: %s", err.Error())
		return nil, err
	}
	for _, instance := range out.DBClusters {
		segments := strings.Split(*instance.EngineVersion, ".")

		eol := ""

		if len(segments) > 0 {
			if cycle, ok := cycleMap[fmt.Sprintf("%s-%s", *instance.Engine, segments[0])]; ok {
				eol = fmt.Sprintf("%v", cycle.EOL)
			}
		}
		if len(segments) > 1 {
			if cycle, ok := cycleMap[fmt.Sprintf("%s-%s.%s", *instance.Engine, segments[0], segments[1])]; ok {
				eol = fmt.Sprintf("%v", cycle.EOL)
			}
		}
		if cycle, ok := cycleMap[fmt.Sprintf("%s-%s", *instance.Engine, *instance.EngineVersion)]; ok {
			eol = fmt.Sprintf("%v", cycle.EOL)
		}

		daysDiff := remainingDays(eol)

		logrus.Debugf("rds cluster: %s -> %s (%s), [%d]", *instance.DBClusterArn, *instance.Engine, *instance.EngineVersion, int(daysDiff))
		rdsClusters = append(rdsClusters, types.RDSCluster{
			Engine: *instance.Engine,
			VersionedResource: types.VersionedResource{
				ID:             *instance.DBClusterIdentifier,
				Kind:           types.KindRDSCluster,
				Parents:        []types.ParentResource{{Kind: types.KindAWSAccount, ID: awsClient.GetAccountId()}},
				Arn:            *instance.DBClusterArn,
				Version:        *instance.EngineVersion,
				CurrentVersion: currentCycleMap[*instance.Engine],
				EOL: types.EOLStatus{
					EOLDate:       eol,
					RemainingDays: daysDiff,
					Status:        eolStatus(daysDiff),
				},
			},
		})
	}
	return &types.InventoryReport{RdsClusters: rdsClusters}, nil
}
