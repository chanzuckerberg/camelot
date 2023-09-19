package github

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
	"github.com/chanzuckerberg/camelot/pkg/util"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var artifacthubCache = cmap.New[HashicorpProviderResponse]()

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
		providers, err := findProviders(*repo.Name, "main", filepath.Join(tempDir, *repo.Name))
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
