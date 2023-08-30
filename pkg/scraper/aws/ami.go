package aws

import (
	"context"
	"time"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/chanzuckerberg/camelot/pkg/scraper/interfaces"
	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
)

func extractAMIs(ctx context.Context, awsClient interfaces.AWSClient) (*types.InventoryReport, error) {
	instances, err := awsClient.ListEC2Instances()
	if err != nil {
		return nil, errors.Wrap(err, "unable to list functions")
	}

	amiMap := map[string][]ec2types.Instance{}

	for _, instance := range instances {
		if _, ok := amiMap[*instance.ImageId]; !ok {
			amiMap[*instance.ImageId] = []ec2types.Instance{}
		}
		amiMap[*instance.ImageId] = append(amiMap[*instance.ImageId], instance)
	}

	if len(amiMap) == 0 {
		return &types.InventoryReport{}, nil
	}

	machineImages := []types.MachineImage{}

	amis := maps.Keys(amiMap)
	images, err := awsClient.DescribeAMIs(amis)
	if err != nil {
		return nil, errors.Wrap(err, "unable to describe amis")
	}

	for _, image := range images {
		if image.ImageId == nil {
			logrus.Debug("AMI is misisng an image id, skipping")
			continue
		}

		if image.CreationDate == nil {
			logrus.Debugf("AMI %s is misisng a creation date, skipping", *image.ImageId)
			continue
		}

		t, err := time.Parse("2006-01-02T15:04:05.000Z0700", *image.CreationDate)
		if err != nil {
			logrus.Debugf("AMI %s has an invalid creation date, skipping", *image.ImageId)
			return nil, err
		}
		endDate := t.AddDate(1, 0, 0).Format("2006-01-02")
		if image.DeprecationTime != nil {
			t, err = time.Parse("2006-01-02T15:04:05.000Z0700", *image.DeprecationTime)
			if err == nil {
				endDate = t.Format("2006-01-02")
			}
		}

		daysDiff := remainingDays(endDate)

		instances, ok := amiMap[*image.ImageId]
		if !ok {
			continue
		}
		for _, instance := range instances {
			machineImages = append(machineImages, types.MachineImage{
				VersionedResource: types.VersionedResource{
					Name:           *image.ImageId,
					Parents:        []types.ParentResource{{Kind: types.KindEC2Instance, ID: *instance.InstanceId}},
					Arn:            "",
					Version:        *image.Name,
					CurrentVersion: "", // TODO: figure out how to find the most recent version of this AMI
					EOL: types.EOLStatus{
						EOLDate:       endDate,
						RemainingDays: daysDiff,
						Status:        eolStatus(daysDiff),
					},
				},
			})
		}
	}

	return &types.InventoryReport{
		MachineImages: machineImages,
	}, nil
}
