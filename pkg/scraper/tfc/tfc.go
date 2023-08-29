package tfc

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/chanzuckerberg/go-misc/errors"
	"github.com/hashicorp/go-tfe"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/sirupsen/logrus"
)

type TFEAssets struct {
	ctx    context.Context
	client *tfe.Client
}

type Repo struct {
	WorkingDir []string `json:"working_dir"`
	Branch     []string `json:"branch"`
	Module     string   `json:"module"`
	Mode       string   `json:"mode"`
	Type       string   `json:"type"`
	Name       string   `json:"name"`
	Provider   string   `json:"provider"`
}

type SourceRepos map[string]*Repo
type Workspaces map[string]SourceRepos
type Orgs map[string]Workspaces

type AWSAssetReference struct {
	// OrgName -> WorkspaceName -> RepoURL -> Repo
	ARN     arn.ARN `json:"arn"`
	TFEOrgs Orgs    `json:"orgs"`
}

type TFEState struct {
	Resources []struct {
		Module    string `json:"module"`
		Mode      string `json:"mode"`
		Type      string `json:"type"`
		Name      string `json:"name"`
		Provider  string `json:"provider"`
		Instances []struct {
			Attributes struct {
				Arn string `json:"arn"`
			} `json:"attributes"`
		} `json:"instances"`
	} `json:"resources"`
}

func mergeWorkspaces(workspaces ...Workspaces) Workspaces {
	merged := Workspaces{}
	for _, workspace := range workspaces {
		for workspaceName, repos := range workspace {
			merged[workspaceName] = repos
		}
	}
	return merged
}

func mergeOrgs(orgs ...Orgs) Orgs {
	merged := Orgs{}
	for _, org := range orgs {
		for orgName, workspaces := range org {
			if _, exists := merged[orgName]; exists {
				merged[orgName] = mergeWorkspaces(merged[orgName], workspaces)
			} else {
				merged[orgName] = workspaces
			}
		}
	}
	return merged
}

func mergeAWSAssets(allAssetMaps []map[string]*AWSAssetReference) map[string]*AWSAssetReference {
	merged := map[string]*AWSAssetReference{}
	for _, assetMap := range allAssetMaps {
		for arn, v := range assetMap {
			if _, exists := merged[arn]; exists {
				merged[arn].TFEOrgs = mergeOrgs(merged[arn].TFEOrgs, v.TFEOrgs)
			} else {
				merged[arn] = v
			}
		}
	}
	return merged
}

func (c *TFEAssets) GetWorkspaceState(ctx context.Context, workspace *tfe.Workspace) (map[string]*AWSAssetReference, error) {
	awsAssets := map[string]*AWSAssetReference{}
	currentState, err := c.client.StateVersions.ReadCurrent(ctx, workspace.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting state versions api for workspace '%s'", workspace.Name)
	}

	response, err := http.Get(currentState.DownloadURL)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading state for workspace '%s'", workspace.Name)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading response body for state for workspace '%s'", workspace.Name)
	}
	// r, err := regexp.Compile("\"(arn:aws:.*?)\"")
	// if err != nil {
	// 	return nil, errors.Wrapf(err, "error compiling regex for workspace '%s'", workspace.Name)
	// }

	// arns := r.FindAllStringSubmatch(string(body), -1)

	var parsedState TFEState
	err = json.Unmarshal(body, &parsedState)
	if err != nil {
		return nil, errors.Wrapf(err, "error unmarshalling state for workspace '%s'", workspace.Name)
	}

	for _, resource := range parsedState.Resources {
		for _, resourceInstance := range resource.Instances {
			if resourceInstance.Attributes.Arn == "" {
				continue
			}
			parsedArn, err := arn.Parse(resourceInstance.Attributes.Arn)
			if err != nil {
				logrus.Debugf("error parsing arn '%s': %s", resourceInstance.Attributes.Arn, err.Error())
				continue
			}
			var vcs map[string]*Repo
			if workspace.VCSRepo != nil {
				vcs = map[string]*Repo{
					workspace.VCSRepo.RepositoryHTTPURL: {
						Branch:     []string{workspace.VCSRepo.Branch},
						WorkingDir: []string{workspace.WorkingDirectory},
						Type:       resource.Type,
						Module:     resource.Module,
						Mode:       resource.Mode,
						Name:       resource.Name,
						Provider:   resource.Provider,
					},
				}
			} else {
				vcs = map[string]*Repo{
					"no-vcs": {
						Type:     resource.Type,
						Module:   resource.Module,
						Mode:     resource.Mode,
						Name:     resource.Name,
						Provider: resource.Provider,
					},
				}
			}
			if _, exists := awsAssets[resourceInstance.Attributes.Arn]; exists {
				awsAssets[resourceInstance.Attributes.Arn].TFEOrgs[workspace.Organization.Name][workspace.Name] = vcs
			} else {
				awsAssets[resourceInstance.Attributes.Arn] = &AWSAssetReference{
					ARN: parsedArn,
					TFEOrgs: Orgs{
						workspace.Organization.Name: {
							workspace.Name: vcs,
						},
					},
				}
			}
		}

	}
	return awsAssets, nil
}

func (c *TFEAssets) GetAllWorkspaceStates(ctx context.Context, orgWorkspaces map[string][]*tfe.Workspace) ([]map[string]*AWSAssetReference, error) {
	awsAssets := []map[string]*AWSAssetReference{}
	for orgName, workspaces := range orgWorkspaces {
		logrus.Debugf("getting workspace states for org %s", orgName)

		var wg sync.WaitGroup
		wg.Add(len(workspaces))
		orgAssets := cmap.New[map[string]*AWSAssetReference]()

		for _, w := range workspaces {
			go func(w *tfe.Workspace, id string) {
				defer wg.Done()
				logrus.Debugf("getting workspace state for workspace %s", w.Name)
				awsAsset, err := c.GetWorkspaceState(ctx, w)
				if err == nil {
					orgAssets.Set(id, awsAsset)
					return
				}

				logrus.Debugf("error getting workspace state for workspace %s: %s", w.Name, err.Error())
			}(w, w.ID)
		}

		wg.Wait()

		for _, v := range orgAssets.Items() {
			awsAssets = append(awsAssets, v)
		}

		wg.Wait()
	}
	return awsAssets, nil
}

func (c *TFEAssets) AllWorkspacesByOrg(ctx context.Context, orgs map[string]*tfe.Organization) (map[string][]*tfe.Workspace, error) {
	orgWorkspace := cmap.New[[]*tfe.Workspace]()

	var wg sync.WaitGroup
	wg.Add(len(orgs))

	for _, v := range orgs {
		logrus.Debugf("getting workspaces for org %s", v.Name)
		go func(org *tfe.Organization) {
			defer wg.Done()
			opts := tfe.WorkspaceListOptions{
				ListOptions: tfe.ListOptions{PageNumber: 1, PageSize: 100},
				Include:     []tfe.WSIncludeOpt{tfe.WSOrganization, tfe.WSCurrentRun},
			}
			items := []*tfe.Workspace{}
			for {
				workspace, err := c.client.Workspaces.List(ctx, org.Name, &opts)
				if err != nil {
					logrus.Debugf("error getting workspaces for org %s: %s", org.Name, err.Error())
					return
				}
				items = append(items, workspace.Items...)
				if workspace.CurrentPage >= workspace.TotalPages {
					break
				}
				opts.PageNumber = workspace.NextPage
			}
			orgWorkspace.Set(org.Name, items)
		}(v)
	}

	wg.Wait()
	res := map[string][]*tfe.Workspace{}
	for k, v := range orgWorkspace.Items() {
		res[k] = v
	}
	return res, nil
}

func (c *TFEAssets) GetAllOrgs(ctx context.Context) (map[string]*tfe.Organization, error) {
	logrus.Debug("Retrieving all TFC organizations")
	opts := tfe.OrganizationListOptions{
		ListOptions: tfe.ListOptions{
			PageNumber: 1,
			PageSize:   100},
	}

	orgMap := map[string]*tfe.Organization{}

	for {
		orgList, err := c.client.Organizations.List(ctx, &opts)
		if err != nil {
			return nil, err
		}

		for _, v := range orgList.Items {
			orgMap[v.Name] = v
		}
		if orgList.CurrentPage >= orgList.TotalPages {
			break
		}
		opts.PageNumber = orgList.NextPage
	}

	return orgMap, nil
}

func (c *TFEAssets) GetAllWorkspaces() (map[string][]*tfe.Workspace, error) {
	orgs, err := c.GetAllOrgs(c.ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting orgs")
	}
	orgWorkspaces, err := c.AllWorkspacesByOrg(c.ctx, orgs)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting workspaces")
	}

	return orgWorkspaces, nil
}

func (c *TFEAssets) GetAllManagedAssets(orgWorkspaces map[string][]*tfe.Workspace) (map[string]*AWSAssetReference, error) {
	AWSAssets, err := c.GetAllWorkspaceStates(c.ctx, orgWorkspaces)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting workspace states")
	}
	mergedAWSAssets := mergeAWSAssets(AWSAssets)
	return mergedAWSAssets, nil
}

// Expects the following evn vars to be set:
// TFE_TOKEN=<secret>
// TFE_ADDRESS=https://<tfe-or-tfc-url>/
func Setup(ctx context.Context) (*TFEAssets, error) {
	client, err := tfe.NewClient(tfe.DefaultConfig())
	if err != nil {
		return nil, err
	}
	return &TFEAssets{
		ctx:    ctx,
		client: client,
	}, nil
}
