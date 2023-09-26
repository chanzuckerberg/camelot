package types

type Status string

const StatusValid = "VALID"
const StatusWarning = "WARNING"
const StatusCritical = "CRITICAL"

type ResourceKind string

const KindAWSAccount ResourceKind = "aws"
const KindEC2Instance ResourceKind = "ec2"
const KindMachineImage ResourceKind = "ami"
const KindRDSCluster ResourceKind = "rds"
const KindVolume ResourceKind = "vol"
const KindLambda ResourceKind = "lambda"
const KindACMCertificate ResourceKind = "cert"
const KindEKSCluster ResourceKind = "eks"
const KindHelmRelease ResourceKind = "helm"
const KindGithubOrg ResourceKind = "github-org"
const KindGithubRepo ResourceKind = "github-repo"
const KindGitPath ResourceKind = "path"
const KindTerrfaormModule ResourceKind = "tf-module"
const KindTFCOrg ResourceKind = "tfc-org"
const KindTFCWorkspace ResourceKind = "tfc-workspace"
const KindTFCResource ResourceKind = "tfc-resource"
const KindTFCProvider ResourceKind = "tfc-provider"

type Versioned interface {
	GetVersionedResource() VersionedResource
}

type EOLStatus struct {
	EOLDate       string `json:"eol_date,omitempty"`
	RemainingDays int    `json:"remaining_active_days"`
	Status        Status `json:"status,omitempty"`
}

type GitOpsReference struct {
	Repo   string `json:"repo,omitempty"`
	Branch string `json:"branch,omitempty"`
	Path   string `json:"path,omitempty"`
}

type ParentResource struct {
	Kind ResourceKind `json:"kind,omitempty"`
	ID   string       `json:"id,omitempty"`
}

type VersionedResource struct {
	Kind            ResourceKind     `json:"kind,omitempty"`
	ID              string           `json:"id,omitempty"`
	Arn             string           `json:"arn,omitempty"`
	Parents         []ParentResource `json:"parents,omitempty"`
	Version         string           `json:"version,omitempty"`
	CurrentVersion  string           `json:"current_version,omitempty"`
	GitOpsReference GitOpsReference  `json:"gitops_reference,omitempty"`
	EOL             EOLStatus        `json:"eol,omitempty"`
}

type EKSCluster struct {
	VersionedResource
	PlatformVersion string            `json:"platform_version,omitempty"`
	Addons          []EKSClusterAddon `json:"addons,omitempty"`
}

func (r EKSCluster) GetVersionedResource() VersionedResource {
	return r.VersionedResource
}

type EKSClusterAddon struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
	Status  string `json:"status,omitempty"`
}

type RDSCluster struct {
	VersionedResource
	Engine string `json:"engine,omitempty"`
}

func (r RDSCluster) GetVersionedResource() VersionedResource {
	return r.VersionedResource
}

type Lambda struct {
	VersionedResource
	Engine string `json:"engine,omitempty"`
}

func (r Lambda) GetVersionedResource() VersionedResource {
	return r.VersionedResource
}

type Volume struct {
	VersionedResource
	VolumeType string `json:"volumetype,omitempty"`
	Size       int32  `json:"size,omitempty"`
}

func (r Volume) GetVersionedResource() VersionedResource {
	return r.VersionedResource
}

type ACMCertificate struct {
	VersionedResource
	InUse            bool     `json:"inuse,omitempty"`
	Status           string   `json:"status,omitempty"`
	Expiration       string   `json:"expiration,omitempty"`
	AutoRenewal      bool     `json:"autorenewal,omitempty"`
	DomainName       string   `json:"domainname,omitempty"`
	AlternativeNames []string `json:"alternativenames,omitempty"`
}

func (r ACMCertificate) GetVersionedResource() VersionedResource {
	return r.VersionedResource
}

type GitRepo struct {
	VersionedResource
}

func (r GitRepo) GetVersionedResource() VersionedResource {
	return r.VersionedResource
}

type TerraformModule struct {
	VersionedResource
}

func (r TerraformModule) GetVersionedResource() VersionedResource {
	return r.VersionedResource
}

type HelmRelease struct {
	VersionedResource
}

func (r HelmRelease) GetVersionedResource() VersionedResource {
	return r.VersionedResource
}

type MachineImage struct {
	VersionedResource
}

func (r MachineImage) GetVersionedResource() VersionedResource {
	return r.VersionedResource
}

type TfcResource struct {
	VersionedResource
}

func (r TfcResource) GetVersionedResource() VersionedResource {
	return r.VersionedResource
}

type TfcWorkspace struct {
	VersionedResource
}

func (r TfcWorkspace) GetVersionedResource() VersionedResource {
	return r.VersionedResource
}

type TfcProvider struct {
	VersionedResource
}

func (r TfcProvider) GetVersionedResource() VersionedResource {
	return r.VersionedResource
}

type Indentity struct {
	AwsAccountNumber string `json:"aws_account_number,omitempty"`
}

type InventoryReport struct {
	Identity  Indentity   `json:"identity,omitempty"`
	Resources []Versioned `json:"resources,omitempty"`
	// EksClusters   []EKSCluster        `json:"eks_clusters,omitempty"`
	// RdsClusters   []RDSCluster        `json:"rds_clusters,omitempty"`
	// Lambdas       []Lambda            `json:"lambdas,omitempty"`
	// Repos         []GitRepo           `json:"repos,omitempty"`
	// Modules       []TerraformModule   `json:"modules,omitempty"`
	// HelmReleases  []HelmRelease       `json:"helm_releases,omitempty"`
	// MachineImages []MachineImage      `json:"machine_images,omitempty"`
	// TfcResources  []TfcResource       `json:"tfc_resources,omitempty"`
	// TfcWorkspaces []TfcWorkspace      `json:"tfc_workspace,omitempty"`
}

type ProductCycle struct {
	Cycle             string      `json:"cycle"`
	ReleaseDate       string      `json:"releaseDate"`
	Support           interface{} `json:"support"` // Could be a bool or a date string: https://endoflife.date/api/nodejs.json
	EOL               interface{} `json:"eol"`     // Could be a bool or a date string
	Latest            string      `json:"latest"`
	LatestReleaseDate string      `json:"latestReleaseDate"`
	Link              string      `json:"link,omitempty"`
	LTS               interface{} `json:"lts"` // Could be a bool or a date string
}
