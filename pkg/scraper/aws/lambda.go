package aws

import (
	"context"
	"fmt"

	lambda_types "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/chanzuckerberg/camelot/pkg/scraper/interfaces"
	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func extractLambdas(ctx context.Context, awsClient interfaces.AWSClient) (*types.InventoryReport, error) {

	cycleMap := map[string]types.ProductCycle{}
	currentCycleMap := map[string]string{}
	products := []string{"python", "ruby", "nodejs"}

	// TODO: Figure out what to do with the go1.x runtime

	for _, product := range products {
		cycles, err := endOfLife(product)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to get %s end of life data", product)
		}

		for _, cycle := range *cycles {
			currentCycleMap[product+cycle.Cycle] = product + (*cycles)[0].Cycle
			currentCycleMap[product+cycle.Cycle+".x"] = product + (*cycles)[0].Cycle

			cycleMap[product+cycle.Cycle] = cycle
			cycleMap[product+cycle.Cycle+".x"] = cycle
		}
	}

	lambdas := []types.Lambda{}
	out, err := awsClient.ListLambdaFunctions()
	if err != nil {
		logrus.Errorf("unable to list functions: %s", err.Error())
		return nil, err
	}
	for _, function := range out.Functions {
		eol := ""
		if cycle, ok := cycleMap[string(function.Runtime)]; ok {
			eol = fmt.Sprintf("%v", cycle.EOL)
		}

		daysDiff := remainingDays(eol)
		version := string(function.Runtime)
		if function.PackageType == lambda_types.PackageTypeImage {
			version = "unversioned"
		}

		logrus.Debugf("lambda function: %s -> %s [%d]", *function.FunctionArn, function.Runtime, daysDiff)
		lambdas = append(lambdas, types.Lambda{
			VersionedResource: types.VersionedResource{
				Name:           *function.FunctionName,
				Parents:        []types.ParentResource{{Kind: "account", ID: awsClient.GetAccountId()}},
				Arn:            *function.FunctionArn,
				Version:        version,
				CurrentVersion: currentCycleMap[string(function.Runtime)],
				EOL: types.EOLStatus{
					EOLDate:       eol,
					RemainingDays: daysDiff,
					Status:        eolStatus(daysDiff),
				},
			},
			Engine: string(function.Runtime),
		})
	}
	return &types.InventoryReport{Lambdas: lambdas}, nil
}
