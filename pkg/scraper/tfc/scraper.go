package tfc

import (
	"context"
	"fmt"
	"time"

	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
	"github.com/chanzuckerberg/camelot/pkg/util"
	"github.com/hashicorp/go-version"
)

func Scrape(ctx context.Context) (*types.InventoryReport, error) {
	report := &types.InventoryReport{}

	tfe_manager, err := Setup(ctx)
	if err != nil {
		return nil, fmt.Errorf("error setting up TFC/TFE Manager: %w", err)
	}

	orgWorkspaces, err := tfe_manager.GetAllWorkspaces()
	if err != nil {
		return nil, fmt.Errorf("error getting TFE/TFC workspaces: %w", err)
	}

	assets, _, err := tfe_manager.GetAllManagedAssets(orgWorkspaces)
	if err != nil {
		return nil, fmt.Errorf("error getting all managed assets: %w", err)
	}

	tfcWorkspaces := []types.TfcWorkspace{}

	for org, workspaces := range orgWorkspaces {
		for _, workspace := range workspaces {
			eolDate := workspace.UpdatedAt.AddDate(0, 3, 0)

			var status types.Status = types.StatusValid
			if time.Now().After(eolDate) {
				status = types.StatusWarning
			}

			resource := types.TfcWorkspace{
				VersionedResource: types.VersionedResource{
					ID:      workspace.Name,
					Kind:    types.KindTFCWorkspace,
					Parents: []types.ParentResource{{Kind: types.KindTFCOrg, ID: org}},
					Version: workspace.TerraformVersion,
					GitOpsReference: types.GitOpsReference{
						Path: workspace.WorkingDirectory,
					},
					EOL: types.EOLStatus{
						EOLDate:       eolDate.Format("2006-01-02"),
						RemainingDays: util.RemainingDays(eolDate),
						Status:        status,
					},
				},
			}

			if workspace.VCSRepo != nil {
				resource.GitOpsReference.Repo = workspace.VCSRepo.Identifier
				resource.GitOpsReference.Branch = workspace.VCSRepo.Branch
			}

			tfcWorkspaces = append(tfcWorkspaces, resource)
		}
	}

	mostPopularVersion, err := getMostPopularTerraformVersion(tfcWorkspaces)
	if err == nil {
		for i, tfcWorkspace := range tfcWorkspaces {
			tfcWorkspaces[i].CurrentVersion = mostPopularVersion.String()
			v, err := version.NewVersion(tfcWorkspace.Version)
			if err == nil && v.LessThan(mostPopularVersion) {
				tfcWorkspaces[i].EOL.Status = types.StatusWarning
			}
		}
	}

	for _, tfcWorkspace := range tfcWorkspaces {
		report.Resources = append(report.Resources, tfcWorkspace)
	}

	for _, asset := range assets {
		if asset.TFEOrgs != nil {
			for orgName, workspaces := range asset.TFEOrgs {
				for workspace, ws := range workspaces {
					for repoUrl, repo := range ws {
						for _, branch := range repo.Branch {
							if branch == "" {
								branch = "main"
							}
							var status types.Status = types.StatusValid
							parents := []types.ParentResource{{Kind: types.KindTFCWorkspace, ID: orgName + "/" + workspace}}
							if len(asset.ARN.AccountID) > 0 {
								parents = append(parents, types.ParentResource{Kind: types.KindAWSAccount, ID: asset.ARN.AccountID})
							}

							report.Resources = append(report.Resources, types.TfcResource{
								VersionedResource: types.VersionedResource{
									ID:      asset.ARN.Service + ":" + asset.ARN.Resource,
									Kind:    types.KindTFCResource,
									Parents: parents,
									GitOpsReference: types.GitOpsReference{
										Repo:   repoUrl,
										Branch: branch,
										Path:   repo.WorkingDir[0],
									},
									EOL: types.EOLStatus{
										Status: status,
									},
								},
							})
						}
					}
				}
			}
		}
	}

	return report, nil
}
