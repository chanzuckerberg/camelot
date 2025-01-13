package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/go-github/v53/github"
	cmap "github.com/orcaman/concurrent-map/v2"
	"golang.org/x/oauth2"
)

var tagCache = cmap.New[time.Time]()

func getTagCommitDate(token, owner, repo, ref string) (*time.Time, error) {
	cacheKey := fmt.Sprintf("%s/%s/%s", owner, repo, ref)
	if t, ok := tagCache.Get(cacheKey); ok {
		return &t, nil
	}

	// if ref looks like a git sha, treat it as such
	re := regexp.MustCompile(`^[0-9a-f]{40}$`)
	if re.MatchString(ref) {
		date, err := getCommitDate(token, owner, repo, ref)
		if err != nil {
			return nil, fmt.Errorf("failed to get commit date: %w", err)
		}
		tagCache.Set(cacheKey, *date)
		return date, nil
	}

	// Treat the reference as a tag
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/ref/tags/%s", owner, repo, ref)
	m, err := getGithubResponse(token, url)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag details: %w", err)
	}

	object, ok := m["object"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to parse object out")
	}
	sha, ok := object["sha"].(string)
	if !ok {
		return nil, fmt.Errorf("failed to parse sha out")
	}

	refType, ok := object["type"].(string)
	if !ok {
		return nil, fmt.Errorf("failed to parse type out")
	}

	// If the tag is a reference to another tag, get the sha of the tag
	if refType == "tag" {
		url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/tags/%s", owner, repo, sha)
		m, err := getGithubResponse(token, url)
		if err != nil {
			return nil, fmt.Errorf("failed to get tag details: %w", err)
		}
		object, ok = m["object"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("failed to parse object out")
		}
		sha, ok = object["sha"].(string)
		if !ok {
			return nil, fmt.Errorf("failed to parse sha out")
		}
	}

	// Query the actual commit details
	date, err := getCommitDate(token, owner, repo, sha)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit date: %w", err)
	}

	tagCache.Set(cacheKey, *date)
	return date, nil
}

func getDefaultBranch(token, owner, repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	m, err := getGithubResponse(token, url)
	if err != nil {
		return "", fmt.Errorf("failed to get repo details: %w", err)
	}

	defaultBranch, ok := m["default_branch"].(string)
	if !ok {
		return "", fmt.Errorf("failed to parse default_branch out")
	}
	return defaultBranch, nil
}

func getCommitDate(token, owner, repo, sha string) (*time.Time, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/commits/%s", owner, repo, sha)
	m, err := getGithubResponse(token, url)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit details: %w", err)
	}

	commit, ok := m["commit"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to parse commit out")
	}
	committer, ok := commit["committer"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to parse committer out")
	}
	dateStr, ok := committer["date"].(string)
	if !ok {
		return nil, fmt.Errorf("failed to parse date out")
	}
	date, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse date string (%s): %w", dateStr, err)
	}
	return &date, nil
}

func getGithubResponse(token, url string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if len(token) > 0 {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to retrieve %s: %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}

	err = json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func parseModuleSource(moduleSource string) (gitUrl string, modulePath string, ref string, err error) {
	parts := strings.Split(moduleSource, "//")
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("invalid module source %s: %w", moduleSource, err)
	}

	gitUrl = parts[0]
	modulePathAndRef := parts[1]

	re := regexp.MustCompile(`\.git$`)
	gitUrl = re.ReplaceAllString(gitUrl, "")

	modulePath = modulePathAndRef
	modulePathAndRefParts := strings.Split(modulePathAndRef, "?ref=")

	if len(modulePathAndRefParts) == 2 {
		modulePath = modulePathAndRefParts[0]
		ref = modulePathAndRefParts[1]
	}

	gitUrl = strings.Replace(gitUrl, "git@github.com:", "github.com/", 1)

	return gitUrl, modulePath, ref, nil
}

func parseGitUrl(gitUrl string) (string, string, error) {
	parts := strings.Split(gitUrl, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid git url %s", gitUrl)
	}

	org := parts[len(parts)-2]
	repo := parts[len(parts)-1]

	return org, repo, nil
}

// Sort module tags by timestamp in a reverse order
func versionSort(moduleRefs []ModuleRef) []ModuleRef {
	sort.Slice(moduleRefs, func(i, j int) bool {
		return moduleRefs[i].Timestamp.After(moduleRefs[j].Timestamp)
	})
	return moduleRefs
}

func getOrgRepos(ctx context.Context, githubToken, githubOrg string) ([]*github.Repository, error) {
	client := github.NewClient(nil)
	if len(githubToken) > 0 {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: githubToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	}
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var allRepos []*github.Repository
	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, githubOrg, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list repos: %w", err)
		}
		for _, repo := range repos {
			if repo.Archived != nil && !*repo.Archived {
				if repo.Language != nil && *repo.Language == "HCL" {
					allRepos = append(allRepos, repo)
				}
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return allRepos, nil
}

var ch = make(chan bool, 20)

func cloneRepo(repoUrl, repoName, destination string) error {
	ch <- true
	defer func() {
		<-ch
	}()

	cmd := exec.Command("git", "clone", "--depth", "1", repoUrl, filepath.Join(destination, repoName))

	if bts, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cannot clone repo %s: %s: %w", repoUrl, string(bts), err)
	}
	return nil
}
