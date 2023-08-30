package tfc

import (
	"context"
	"time"

	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
	"github.com/chanzuckerberg/camelot/pkg/util"
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
)

func Scrape() (*types.InventoryReport, error) {
	report := &types.InventoryReport{}

	tfe_manager, err := Setup(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "error setting up TFC/TFE Manager")
	}

	orgWorkspaces, err := tfe_manager.GetAllWorkspaces()
	if err != nil {
		return nil, errors.Wrap(err, "error getting TFE/TFC workspaces")
	}

	assets, err := tfe_manager.GetAllManagedAssets(orgWorkspaces)
	if err != nil {
		return nil, errors.Wrap(err, "error getting all managed assets")
	}

	for org, workspaces := range orgWorkspaces {
		for _, workspace := range workspaces {
			eolDate := workspace.UpdatedAt.AddDate(0, 3, 0)

			var status types.Status = types.StatusActive
			if time.Now().After(eolDate) {
				status = types.StatusOutdated
			}

			resource := types.TfcWorkspace{
				VersionedResource: types.VersionedResource{
					Name:    workspace.Name,
					Parents: []types.ParentResource{{Kind: "tfc-org", ID: org}},
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

			report.TfcWorkspaces = append(report.TfcWorkspaces, resource)
		}
	}

	mostPopularVersion, err := getMostPopularTerraformVersion(report.TfcWorkspaces)

	if err == nil {
		for i, tfcWorkspace := range report.TfcWorkspaces {
			v, err := version.NewVersion(tfcWorkspace.Version)
			if err == nil && v.LessThan(mostPopularVersion) {
				report.TfcWorkspaces[i].EOL.Status = types.StatusOutdated
			}
		}
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
							var status types.Status = types.StatusActive
							parents := []types.ParentResource{{Kind: "tfc-workspace", ID: orgName + "/" + workspace}}
							if len(asset.ARN.AccountID) > 0 {
								parents = append(parents, types.ParentResource{Kind: "account", ID: asset.ARN.AccountID})
							}

							report.TfcResources = append(report.TfcResources, types.TfcResource{
								VersionedResource: types.VersionedResource{
									Name:    asset.ARN.Service + ":" + asset.ARN.Resource,
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
