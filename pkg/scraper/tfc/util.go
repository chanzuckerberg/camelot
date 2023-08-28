package tfc

import (
	"sort"

	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
)

func getMostPopularTerraformVersion(workspaces []types.TfcWorkspace) (*version.Version, error) {
	versionDistribution := map[string]int{}
	for _, tfcWorkspace := range workspaces {
		if count, ok := versionDistribution[tfcWorkspace.Version]; !ok {
			versionDistribution[tfcWorkspace.Version] = 1
		} else {
			versionDistribution[tfcWorkspace.Version] = count + 1
		}
	}

	terraformVersions := make([]string, 0, len(versionDistribution))
	for version := range versionDistribution {
		terraformVersions = append(terraformVersions, version)
	}

	if len(terraformVersions) > 0 {
		sort.SliceStable(terraformVersions, func(i, j int) bool {
			return versionDistribution[terraformVersions[i]] >= versionDistribution[terraformVersions[j]]
		})
		return version.NewVersion(terraformVersions[0])
	}
	return nil, errors.New("no terraform versions found")
}
