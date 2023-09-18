package github

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/pkg/errors"
)

var providerBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "terraform",
		},
		{
			Type: "required_providers",
		},
		{
			Type:       "provider",
			LabelNames: []string{"name"},
		},
	},
}

type HashicorpProviderResponse struct {
	ID          string    `json:"id"`
	Owner       string    `json:"owner"`
	Namespace   string    `json:"namespace"`
	Name        string    `json:"name"`
	Alias       string    `json:"alias"`
	Version     string    `json:"version"`
	Tag         string    `json:"tag"`
	Description string    `json:"description"`
	Source      string    `json:"source"`
	PublishedAt time.Time `json:"published_at"`
	Downloads   int       `json:"downloads"`
	Tier        string    `json:"tier"`
	LogoURL     string    `json:"logo_url"`
	Versions    []string  `json:"versions"`
}

func findProviders(repo, branch, dir string) ([]types.Versioned, error) {
	providers := []types.Versioned{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".terraform" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".tf" {
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		f, diags := hclsyntax.ParseConfig(b, path, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			return errors.Wrapf(diags.Errs()[0], "failed to parse %s", path)
		}

		content, _, diags := f.Body.PartialContent(providerBlockSchema)
		if diags.HasErrors() {
			return errors.Wrap(diags.Errs()[0], "terraform code has errors")
		}

		for _, block := range content.Blocks {
			if block.Type != "terraform" {
				continue
			}

			content, _ := block.Body.Content(providerBlockSchema)
			for _, innerBlock := range content.Blocks {
				if innerBlock.Type != "required_providers" {
					continue
				}

				attrs, _ := innerBlock.Body.JustAttributes()
				for name, attr := range attrs {
					attrVal, diag := attr.Expr.Value(nil)
					if diag != nil {
						continue
					}
					versionConstraint := ""
					providerID := fmt.Sprintf("hashicorp/%s", name)
					eol := types.EOLStatus{
						Status: types.StatusValid,
					}
					if attrVal.Type().IsPrimitiveType() {
						// Legacy version reference
						eol.Status = types.StatusCritical
					} else {
						if v, ok := attrVal.AsValueMap()["version"]; ok {
							versionConstraint = v.AsString()
						}
						if v, ok := attrVal.AsValueMap()["source"]; ok {
							providerID = v.AsString()
						}
					}

					provider, err := getProviderDetails(providerID)
					mostCurrentVer := ""
					if err == nil {
						mostCurrentVer = provider.Version
					}

					if !checkProviderVersion(versionConstraint, mostCurrentVer) {
						eol.Status = types.StatusCritical
					}

					relativePath := strings.TrimPrefix(path, dir+"/")
					providers = append(providers, types.TfcProvider{
						VersionedResource: types.VersionedResource{
							ID:             providerID,
							Kind:           types.KindTFCProvider,
							Version:        versionConstraint,
							CurrentVersion: mostCurrentVer,
							Parents:        []types.ParentResource{{Kind: types.KindGithubRepo, ID: repo}, {Kind: types.KindGitPath, ID: relativePath}},
							GitOpsReference: types.GitOpsReference{
								Repo:   repo,
								Branch: branch,
								Path:   relativePath,
							},
							EOL: eol,
						},
					})
				}
			}
		}
		return nil
	})
	return providers, err
}

func getProviderDetails(providerID string) (*HashicorpProviderResponse, error) {
	if val, ok := artifacthubCache.Get(providerID); ok {
		return &val, nil
	}

	p, err := url.Parse(fmt.Sprintf("https://registry.terraform.io/v1/providers/%s", providerID))
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse endpoint url")
	}

	req, err := http.NewRequest("GET", p.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "Http Get call to terraform registry failed")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, errors.Errorf("unable to query the search api, got %s", res.Status)
	}

	result := HashicorpProviderResponse{}

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode response")
	}

	artifacthubCache.Set(providerID, result)
	return &result, nil
}

func checkProviderVersion(verConstraint, mostCurrentVer string) bool {
	if verConstraint == "" || mostCurrentVer == "" {
		return true
	}

	constraints, err := version.NewConstraint(verConstraint)
	if err != nil {
		return false
	}

	if constraints.Len() == 0 {
		return false
	}

	mostCurrentVersion := version.Must(version.NewVersion(mostCurrentVer))
	valid := constraints.Check(mostCurrentVersion) // Current version no longer satisfies the constraint
	if !valid {
		return false
	}

	oldestSupportedVersion, err := findOldestVersionConstraint(verConstraint)
	if err != nil {
		return false
	}

	if !isVersionSupported(oldestSupportedVersion, mostCurrentVersion) {
		return false // Current version satisfies the constraint, but is too old
	}

	return true
}

func isVersionSupported(v, mostCurrentVer *version.Version) bool {
	majorVer := v.Segments()[0]
	majorCurrentVer := mostCurrentVer.Segments()[0]
	if majorCurrentVer >= 2 {
		if majorVer < majorCurrentVer-2 {
			return false
		}
	}
	return true
}
