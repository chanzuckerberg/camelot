package aws

import (
	"context"

	"github.com/chanzuckerberg/camelot/pkg/scraper/interfaces"
	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
	"github.com/pkg/errors"
)

func extractVolumes(ctx context.Context, awsClient interfaces.AWSClient) (*types.InventoryReport, error) {
	out, err := awsClient.ListVolumes()
	if err != nil {
		return nil, errors.Wrap(err, "unable to list volumes")
	}
	volumes := []types.Versioned{}
	for _, volume := range out {
		volumes = append(volumes, types.Volume{
			VolumeType: string(volume.VolumeType),
			Size:       *volume.Size,
			VersionedResource: types.VersionedResource{
				ID:      *volume.VolumeId,
				Kind:    types.KindVolume,
				Parents: []types.ParentResource{{Kind: types.KindAWSAccount, ID: awsClient.GetAccountId()}},
				Version: string(volume.VolumeType),
				EOL: types.EOLStatus{
					EOLDate:       "",
					RemainingDays: 9999,
					Status:        types.StatusValid,
				},
			},
		})
	}
	return &types.InventoryReport{
		Resources: volumes,
	}, nil
}
