package github

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseModuleSource(t *testing.T) {
	r := require.New(t)

	url, path, ref, err := parseModuleSource("git@github.com:aws-ia/terraform-aws-eks-blueprints.git//modules/kubernetes-addons?ref=v4.24.0")
	r.NoError(err)
	r.Equal("github.com/aws-ia/terraform-aws-eks-blueprints", url)
	r.Equal("modules/kubernetes-addons", path)
	r.Equal("v4.24.0", ref)

	url, path, ref, err = parseModuleSource("github.com/org/repo//module?ref=v0.43.1")
	r.NoError(err)
	r.Equal("github.com/org/repo", url)
	r.Equal("module", path)
	r.Equal("v0.43.1", ref)
}

func TestParseModuleSourceNoRef(t *testing.T) {
	r := require.New(t)
	gitUrl, modulePath, ref, err := parseModuleSource("github.com/org/repo//terraform/modules/module")
	r.NoError(err)
	r.Equal("github.com/org/repo", gitUrl)
	r.Equal("terraform/modules/module", modulePath)
	r.Equal("", ref)
}

// Testing a tag-to-commit reference
func TestGithubTag(t *testing.T) {
	r := require.New(t)
	m, err := getTagCommitDate("", "google", "go-github", "v53.2.0")
	r.NoError(err)
	r.NotNil(m)
}

// Testing a tag-to-tag-to-commit reference
func TestGithubTag2(t *testing.T) {
	r := require.New(t)
	m, err := getTagCommitDate("", "aws-ia", "terraform-aws-eks-blueprints", "v4.32.1")
	r.NoError(err)
	r.NotNil(m)
}

// Testing a commit reference
func TestGithubSha(t *testing.T) {
	r := require.New(t)
	m, err := getTagCommitDate("", "aws-ia", "terraform-aws-eks-blueprints", "a1de62c0496c6149d67b817ead6519823948d645")
	r.NoError(err)
	r.NotNil(m)
}

func TestCloneRepo(t *testing.T) {
	r := require.New(t)
	tempDir, err := os.MkdirTemp("/tmp", "terraform-aws-eks-blueprints")
	r.NoError(err)
	defer os.RemoveAll(tempDir)
	err = cloneRepo("https://github.com/aws-ia/terraform-aws-eks-blueprints", "terraform-aws-eks-blueprints", tempDir)
	r.NoError(err)
}

func TestGetRepos(t *testing.T) {
	r := require.New(t)
	repos, err := getOrgRepos(context.Background(), "", "aws-ia")
	r.NoError(err)
	r.NotNil(repos)
	r.NotEmpty(repos)
}

func TestGetProviderDetails(t *testing.T) {
	r := require.New(t)
	p, err := getProviderDetails("hashicorp/aws")
	r.NoError(err)
	r.Equal("hashicorp", p.Owner)
	r.Equal("aws", p.Name)
	r.Equal("terraform-provider-aws", p.Description)
	r.Equal("official", p.Tier)
	r.NotEmpty(p.Version)
}

func TestVersionConstraint(t *testing.T) {
	r := require.New(t)
	res := checkProviderVersion("1.0.0", "1.0.0")
	r.True(res)

	res = checkProviderVersion(">= 1.0.0", "1.0.0")
	r.True(res)

	res = checkProviderVersion("~> 1.0.0", "1.0.0")
	r.True(res)

	res = checkProviderVersion("~> 1.0.0", "1.0.1")
	r.True(res)

	res = checkProviderVersion("~> 1.0.0", "1.1.0")
	r.False(res)

	res = checkProviderVersion("~> 1.0.0", "2.0.5")
	r.False(res)

	res = checkProviderVersion(">= 1.0.0", "2.0.5")
	r.True(res)

	res = checkProviderVersion(">= 1.0.0", "2.0.5")
	r.True(res)

	res = checkProviderVersion(">= 1.0.0", "3.0.5")
	r.True(res)

	res = checkProviderVersion(">= 1.0.0", "4.0.0")
	r.False(res)

	res = checkProviderVersion(">= 1.0.0", "7.0.0")
	r.False(res)

	res = checkProviderVersion("~> 3.4", "4.0.0")
	r.False(res)
}
