package types

type Status string

const StatusActive = "ACTIVE"
const StatusEndOfLife = "ENDOFLIFE"
const StatusWarning = "WARNING"
const StatusCritical = "CRITICAL"
const StatusOutdated = "OUTDATED"

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
	Kind string `json:"kind,omitempty"`
	ID   string `json:"name,omitempty"`
}

type VersionedResource struct {
	Name            string           `json:"name,omitempty"`
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

type EKSClusterAddon struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
	Status  string `json:"status,omitempty"`
}

type RDSCluster struct {
	VersionedResource
	Engine string `json:"engine,omitempty"`
}

type Lambda struct {
	VersionedResource
}

type Repo struct {
	VersionedResource
}

type Module struct {
	VersionedResource
}

type HelmRelease struct {
	VersionedResource
}

type MachineImage struct {
	VersionedResource
}

type TfcResource struct {
	VersionedResource
}

type TfcWorkspace struct {
	VersionedResource
}

type Indentity struct {
	AwsAccountNumber string `json:"aws_account_number,omitempty"`
	AwsProfile       string `json:"aws_profile,omitempty"`
}

type InventoryReport struct {
	Identity      Indentity      `json:"identity,omitempty"`
	EksClusters   []EKSCluster   `json:"eks_clusters,omitempty"`
	RdsClusters   []RDSCluster   `json:"rds_clusters,omitempty"`
	Lambdas       []Lambda       `json:"lambdas,omitempty"`
	Repos         []Repo         `json:"repos,omitempty"`
	Modules       []Module       `json:"modules,omitempty"`
	HelmReleases  []HelmRelease  `json:"helm_releases,omitempty"`
	MachineImages []MachineImage `json:"machine_images,omitempty"`
	TfcResources  []TfcResource  `json:"tfc_resources,omitempty"`
	TfcWorkspaces []TfcWorkspace `json:"tfc_workspace,omitempty"`
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
