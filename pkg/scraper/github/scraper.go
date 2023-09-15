package github

import (
	"context"
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
	"github.com/chanzuckerberg/camelot/pkg/util"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var artifacthubCache = cmap.New[HashicorpProviderResponse]()

var moduleBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       "module",
			LabelNames: []string{"name"},
		},
	},
}

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

type ModuleRef struct {
	Ref       string
	Timestamp time.Time
}

func Scrape(githubOrg string) (*types.InventoryReport, error) {
	ctx := context.Background()

	githubToken := os.Getenv("GITHUB_TOKEN")
	allRepos, err := getOrgRepos(ctx, githubToken, githubOrg)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to retrieve github repos for the org")
	}

	if len(allRepos) == 0 {
		return nil, errors.New("No repos found")
	}

	report := &types.InventoryReport{
		Resources: []types.Versioned{},
	}

	moduleUsageMap := map[string]map[string]int{}
	repoModuleReferenceMap := map[string]map[string]bool{}

	for _, repo := range allRepos {
		tempDir, err := os.MkdirTemp("/tmp", *repo.Name)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to create temp directory")
		}
		defer os.RemoveAll(tempDir)

		err = cloneRepo(*repo.CloneURL, *repo.Name, tempDir)
		if err != nil {
			return nil, errors.Wrap(err, "Unable to clone repo")
		}
		providers, err := findProviders(*repo.Name, "main", tempDir)
		if err == nil {
			report.Resources = append(report.Resources, providers...)
		} else {
			logrus.Debugf("Unable to read providers in %s: %s", *repo.Name, err.Error())
		}
		modules, err := findModules(tempDir)
		if err != nil {
			logrus.Debugf("Unable to read modules in %s: %s", *repo.Name, err.Error())
			continue
		}
		for _, module := range modules {
			// Only track versioned module references
			if !strings.Contains(module, "?ref=") {
				continue
			}
			gitUrl, modulePath, ref, err := parseModuleSource(module)
			if err != nil {
				logrus.Errorf("Unable to parse module source %s in repo %s: %s", module, *repo.Name, err.Error())
				continue
			}

			moduleReference := gitUrl + "//" + modulePath
			if _, ok := moduleUsageMap[moduleReference]; !ok {
				moduleUsageMap[moduleReference] = map[string]int{}
			}

			versionedModuleReference := fmt.Sprintf("%s?ref=%s", moduleReference, ref)
			if _, ok := repoModuleReferenceMap[versionedModuleReference]; !ok {
				repoModuleReferenceMap[versionedModuleReference] = map[string]bool{}
			}
			moduleUsageMap[moduleReference][ref] = moduleUsageMap[moduleReference][ref] + 1
			repoModuleReferenceMap[versionedModuleReference][*repo.Name] = true
			logrus.Debugf("module: repo=%s, name=%s, ref=%s", gitUrl, modulePath, ref)
		}

		defaultBranch, err := getDefaultBranch(githubToken, githubOrg, *repo.Name)
		if err != nil {
			logrus.Errorf("Unable to get default branch for %s: %s", *repo.Name, err.Error())
			continue
		}
		date, err := getCommitDate(githubToken, githubOrg, *repo.Name, defaultBranch)
		if err != nil {
			logrus.Errorf("Unable to get commit date for %s: %s", *repo.Name, err.Error())
			date = &time.Time{}
			continue
		}

		eolDate := date.AddDate(3, 0, 0)
		var status types.Status = types.StatusValid
		if time.Now().After(eolDate) {
			status = types.StatusWarning
		}
		report.Resources = append(report.Resources, types.GitRepo{
			VersionedResource: types.VersionedResource{
				ID:      *repo.Name,
				Kind:    types.KindGithubRepo,
				Arn:     "",
				Parents: []types.ParentResource{{Kind: types.KindGithubOrg, ID: githubOrg}},
				Version: "0.0.0",
				EOL: types.EOLStatus{
					EOLDate:       eolDate.Format("2006-01-02"),
					RemainingDays: util.RemainingDays(eolDate),
					Status:        status,
				},
			},
		})
	}

	for module, moduleVersionDistribution := range moduleUsageMap {
		logrus.Debugf("%s:\n", module)

		moduleRefs := []ModuleRef{}
		for ref := range moduleVersionDistribution {
			if ref == "master" || ref == "main" || ref == "" {
				continue
			}

			gitUrl, _, _, err := parseModuleSource(module)
			if err != nil {
				logrus.Errorf("Unable to parse module source %s: %s", module, err.Error())
				continue
			}
			org, repo, err := parseGitUrl(gitUrl)
			if err != nil {
				logrus.Errorf("Unable to parse git url %s: %s", gitUrl, err.Error())
				continue
			}
			timestamp, err := getTagCommitDate(githubToken, org, repo, ref)
			if err != nil {
				logrus.Errorf("Unable to get timestamp for %s: %s", ref, err.Error())
				timestamp = &time.Time{}
			}
			moduleRefs = append(moduleRefs, ModuleRef{
				Ref:       ref,
				Timestamp: *timestamp,
			})
		}

		moduleRefs = versionSort(moduleRefs)

		for _, ref := range moduleRefs {
			logrus.Debugf("\t%s\t%d\n", ref, moduleVersionDistribution[ref.Ref])
		}

		for index, ref := range moduleRefs {
			repos := repoModuleReferenceMap[fmt.Sprintf("%s?ref=%s", module, ref.Ref)]
			var status types.Status
			status = types.StatusWarning
			eolDate := ref.Timestamp.Format("2006-01-02")
			if index == 0 {
				status = types.StatusValid
				// Assume modules are supported for 3 years
				eolDate = ref.Timestamp.AddDate(3, 0, 0).Format("2006-01-02")
			}
			for repo := range repos {
				report.Resources = append(report.Resources, types.TerraformModule{
					VersionedResource: types.VersionedResource{
						ID:             strings.Replace(module, fmt.Sprintf("github.com/%s/", githubOrg), "", 1),
						Kind:           types.KindTerrfaormModule,
						Arn:            "",
						Parents:        []types.ParentResource{{Kind: types.KindGithubRepo, ID: repo}},
						Version:        ref.Ref,
						CurrentVersion: moduleRefs[0].Ref,
						EOL: types.EOLStatus{
							EOLDate:       eolDate,
							RemainingDays: 0,
							Status:        status,
						},
					},
				})
			}
		}

	}

	return report, nil
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
					mostCurrentVer := versionConstraint
					if err == nil {
						mostCurrentVer = provider.Version
					}

					if versionConstraint != "" {
						constraint, err := version.NewConstraint(versionConstraint)
						mostCurrentVersion := version.Must(version.NewVersion(mostCurrentVer))
						if err == nil {
							if !constraint.Check(mostCurrentVersion) {
								eol.Status = types.StatusCritical
								fmt.Printf("FAIL CURRENT: %s %s %s\n", providerID, versionConstraint, mostCurrentVer)
							} else {
								fmt.Printf("OK: %s %s %s\n", providerID, versionConstraint, mostCurrentVer)
								majorVer := mostCurrentVersion.Segments()[0]
								if majorVer > 2 {
									lowestSupportedVer := version.Must(version.NewVersion(fmt.Sprintf("%d.0.0", majorVer-2)))
									if !constraint.Check(lowestSupportedVer) {
										eol.Status = types.StatusCritical
										fmt.Printf("FAIL OLD: %s %s %s\n", providerID, versionConstraint, mostCurrentVer)
									}
								}
							}
						}
					}

					providers = append(providers, types.TfcProvider{
						VersionedResource: types.VersionedResource{
							ID:             providerID,
							Kind:           types.KindTFCProvider,
							Version:        versionConstraint,
							CurrentVersion: mostCurrentVer,
							Parents:        []types.ParentResource{{Kind: types.KindGithubRepo, ID: repo}},
							GitOpsReference: types.GitOpsReference{
								Repo:   repo,
								Branch: branch,
								Path:   strings.TrimPrefix(path, dir+"/"),
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

func findModules(dir string) ([]string, error) {
	modules := []string{}
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

		content, _, diags := f.Body.PartialContent(moduleBlockSchema)
		if diags.HasErrors() {
			return errors.Wrap(diags.Errs()[0], "terraform code has errors")
		}

		for _, block := range content.Blocks {
			if block.Type != "module" {
				continue
			}
			attrs, diags := block.Body.JustAttributes()
			if diags.HasErrors() {
				return errors.Wrap(diags.Errs()[0], "terraform code has errors")
			}
			sourceAttr, ok := attrs["source"]
			if !ok {
				// Module without a source
				continue
			}

			source, diags := sourceAttr.Expr.(*hclsyntax.TemplateExpr).Parts[0].Value(nil)
			if diags.HasErrors() {
				return errors.Wrap(diags.Errs()[0], "terraform code has errors")
			}
			modules = append(modules, source.AsString())
		}
		return nil
	})
	return modules, errors.Wrap(err, "failed to walk directory")
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
	constraint, err := version.NewConstraint(verConstraint)
	if err != nil {
		return false
	}
	mostCurrentVersion := version.Must(version.NewVersion(mostCurrentVer))
	valid := constraint.Check(mostCurrentVersion)
	if !valid {
		return false
	}

	majorVer := mostCurrentVersion.Segments()[0]
	if majorVer >= 2 {
		lowestSupportedVer := findLowestSupportedVersion(constraint)
		majorLowestVer := lowestSupportedVer.Segments()[0]
		if majorLowestVer < majorVer-2 {
			return false
		}
	}

	return true
}

func findLowestSupportedVersion(constraint version.Constraints) *version.Version {
	for maj := 0; maj < 1000; maj++ {

		ver := version.Must(version.NewVersion(fmt.Sprintf("%d.0.0", maj)))
		if constraint.Check(ver) {
			return ver
		}

	}
	return nil
}
