package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
)

var artifacthubCache = cmap.New[ArtifactHubSearchResults]()

func getHelmReleases(ctx context.Context, config *rest.Config, namespaces []string, clusterName string) ([]types.HelmRelease, error) {
	helmReleases := []types.HelmRelease{}

	for _, namespace := range namespaces {
		helmClient, err := getHelmClient(config, namespace)
		if err != nil {
			logrus.Debugf("unable to create helm client: %s", err.Error())
			continue
		}
		releases, err := helmClient.ListDeployedReleases()
		if err != nil {
			logrus.Debugf("unable to list helm releases: %s", err.Error())
			continue
		}
		for _, release := range releases {
			if len(release.Chart.Metadata.Home) == 0 {
				continue
			}

			activeVersion := parseChartVersion(release.Chart.Metadata.Version)
			var newestVersion *semver.Version

			// No exact matches can be derived from artifacthub, these are all best guesses, which can produce multiple results
			charts, err := findHelmChartsByName(release.Chart.Metadata.Name)
			if err != nil {
				logrus.Debugf("unable to find matching helm charts: %s", err.Error())
				continue
			}

			var status types.Status = types.StatusValid

			if len(charts) == 1 {
				// Only one match found, use it
				newestVersion = parseChartVersion(charts[0].Version)
			}

			if newestVersion == nil && len(charts) > 0 {
				filteredCharts := filterHelmChartsByHome(charts, release.Chart.Metadata.Home)
				if len(filteredCharts) == 1 {
					// Found the exact match
					newestVersion = parseChartVersion(filteredCharts[0].Version)
				}

				if newestVersion == nil {
					// Found multiple matches, use the most popular one (they are sorted by stars)
					logrus.Debugf("release: %s; version: %s, home: %s, matches: %d", release.Chart.Metadata.Name, release.Chart.Metadata.Version, release.Chart.Metadata.Home, len(charts))
					for _, chart := range charts {
						logrus.Debugf("  chart: %s, stars: %d, orgs: %d, repo: %s", chart.Version, chart.Stars, chart.ProductionOrganizationsCount, chart.Repository.Url)
						newestVersion = parseChartVersion(chart.Version)
						break
					}
					logrus.Debugln()
				}
			}

			if newestVersion != nil {
				if activeVersion.Major() < newestVersion.Major() {
					status = types.StatusCritical
				} else if activeVersion.Minor() < newestVersion.Minor() {
					status = types.StatusWarning
				}
			}

			currentVersionStr := ""
			if newestVersion != nil {
				currentVersionStr = newestVersion.String()
			}
			helmReleases = append(helmReleases, types.HelmRelease{
				VersionedResource: types.VersionedResource{
					ID:             fmt.Sprintf("%s/%s", namespace, release.Name),
					Kind:           types.KindHelmRelease,
					Arn:            "",
					Parents:        []types.ParentResource{{Kind: types.KindEKSCluster, ID: clusterName}},
					Version:        activeVersion.String(),
					CurrentVersion: currentVersionStr,
					EOL: types.EOLStatus{
						EOLDate:       "",
						RemainingDays: 0,
						Status:        status,
					},
				},
			})
		}
	}
	return helmReleases, nil
}

func parseChartVersion(version string) *semver.Version {
	return semver.MustParse(strings.Replace(version, "v", "", 1))
}

func filterHelmChartsByHome(charts []ArtifactHubPackage, home string) []ArtifactHubPackage {
	matchingCharts := []ArtifactHubPackage{}
	for _, chart := range charts {
		if aliasedUrls(chart.Repository.Url, home) {
			matchingCharts = append(matchingCharts, chart)
		}
	}
	return matchingCharts
}

func aliasedUrls(url1, url2 string) bool {
	fuzzyMatchList := []string{"bitnami", "datadoghq.com", "fluent", "rancher", "linkerd", "actions-runner-controller", "uswitch"}
	for _, fuzzyMatch := range fuzzyMatchList {
		if strings.Contains(url1, fuzzyMatch) && strings.Contains(url2, fuzzyMatch) {
			return true
		}
	}

	o1, r1, err1 := extractOrgAndRepo(url1)
	o2, r2, err2 := extractOrgAndRepo(url2)
	if err1 != nil || err2 != nil {
		return false
	}
	return o1 == o2 && r1 == r2
}

func extractOrgAndRepo(url string) (string, string, error) {
	url = strings.ToLower(strings.ReplaceAll(url, "https://", ""))
	if strings.HasPrefix(url, "github.com/") {
		parts := strings.Split(url, "/")
		if len(parts) < 3 {
			return "", "", errors.Errorf("unable to parse github.com url: %s", url)
		}
		return parts[1], parts[2], nil
	}
	if strings.Contains(url, "github.io/") {
		parts := strings.Split(url, "/")
		if len(parts) < 2 {
			return "", "", errors.Errorf("unable to parse github.io url: %s", url)
		}
		return strings.ReplaceAll(parts[0], ".github.io", ""), parts[1], nil
	}
	return "", "", errors.Errorf("unable to parse url: %s", url)
}

func findHelmChartsByName(chartName string) ([]ArtifactHubPackage, error) {
	if val, ok := artifacthubCache.Get(chartName); ok {
		return val.Packages, nil
	}
	query := url.Values{}
	query.Add("ts_query_web", chartName)
	query.Add("kind", "0") // helm charts only
	query.Add("limit", "20")
	query.Add("offset", "0")
	query.Add("deprecated", "false")
	query.Add("sort", "stars")

	p, err := url.Parse("https://artifacthub.io/api/v1/packages/search")
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse endpoint url")
	}

	p.RawQuery = query.Encode()

	req, err := http.NewRequest("GET", p.String(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "Http Get call to the artifacthub api failed")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, errors.Errorf("unable to query the search api, got %s", res.Status)
	}

	result := ArtifactHubSearchResults{}

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return nil, err
	}
	packages := []ArtifactHubPackage{}
	for _, pkg := range result.Packages {
		if pkg.NormalizedName == chartName {
			packages = append(packages, pkg)
		}
	}
	result.Packages = packages
	artifacthubCache.Set(chartName, result)
	return packages, nil
}

type ArtifactHubSearchResults struct {
	Packages []ArtifactHubPackage `json:"packages"`
}

type ArtifactHubPackage struct {
	PackageId                    string                `json:"package_id"`
	Name                         string                `json:"name"`
	NormalizedName               string                `json:"normalized_name"`
	LogoImageId                  string                `json:"logo_image_id"`
	Stars                        int                   `json:"stars"`
	Description                  string                `json:"description"`
	Version                      string                `json:"version"`
	AppVersion                   string                `json:"app_version"`
	Deprecated                   bool                  `json:"deprecated"`
	Signed                       bool                  `json:"signed"`
	ProductionOrganizationsCount int                   `json:"production_organizations_count"`
	Ts                           int                   `json:"ts"`
	Repository                   ArtifactHubRepository `json:"repository"`
}

type ArtifactHubRepository struct {
	Url                     string `json:"url"`
	Kind                    int    `json:"kind"`
	Name                    string `json:"name"`
	Official                bool   `json:"official"`
	DisplayName             string `json:"display_name"`
	RepositoryId            string `json:"repository_id"`
	ScannerDisabled         bool   `json:"scanner_disabled"`
	OrganizationName        string `json:"organization_name"`
	VerifiedPublisher       bool   `json:"verified_publisher"`
	OrganizationDisplayName string `json:"organization_display_name"`
}
