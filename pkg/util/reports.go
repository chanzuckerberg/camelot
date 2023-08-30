package util

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
)

type ReportFilter struct {
	ResourceKinds []types.ResourceKind // TODO: Support logical operators
	ParentKinds   []types.ResourceKind
	ParentIDs     []string
	IDs           []string
	Status        []types.Status
	Version       string
}

func ReportToTable(report types.InventoryReport) [][]string {
	var table [][]string
	for _, item := range report.MachineImages {
		table = append(table, versionedResourceToTableRow(item.VersionedResource))
	}
	for _, item := range report.RdsClusters {
		table = append(table, versionedResourceToTableRow(item.VersionedResource))
	}
	for _, item := range report.EksClusters {
		table = append(table, versionedResourceToTableRow(item.VersionedResource))
	}
	for _, item := range report.Lambdas {
		table = append(table, versionedResourceToTableRow(item.VersionedResource))
	}
	for _, item := range report.Repos {
		table = append(table, versionedResourceToTableRow(item.VersionedResource))
	}
	for _, item := range report.Modules {
		table = append(table, versionedResourceToTableRow(item.VersionedResource))
	}
	for _, item := range report.HelmReleases {
		table = append(table, versionedResourceToTableRow(item.VersionedResource))
	}
	for _, item := range report.TfcResources {
		table = append(table, versionedResourceToTableRow(item.VersionedResource))
	}
	for _, item := range report.TfcWorkspaces {
		table = append(table, versionedResourceToTableRow(item.VersionedResource))
	}
	return table
}

func versionedResourceToTableRow(item types.VersionedResource) []string {
	var sb strings.Builder
	for _, p := range item.Parents {
		if sb.Len() > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(string(p.Kind))
		sb.WriteString(":")
		sb.WriteString(p.ID)
	}
	return []string{
		string(item.Kind),
		truncate(item.ID, 80),
		truncate(sb.String(), 80),
		item.Version,
		item.CurrentVersion,
		string(item.EOL.Status),
		item.EOL.EOLDate,
	}
}

func CombineReports(reports []*types.InventoryReport) types.InventoryReport {
	summary := types.InventoryReport{}
	for _, report := range reports {
		if report == nil {
			continue
		}
		summary.EksClusters = append(summary.EksClusters, report.EksClusters...)
		summary.RdsClusters = append(summary.RdsClusters, report.RdsClusters...)
		summary.Lambdas = append(summary.Lambdas, report.Lambdas...)
		summary.HelmReleases = append(summary.HelmReleases, report.HelmReleases...)
		summary.MachineImages = append(summary.MachineImages, report.MachineImages...)
		summary.TfcResources = append(summary.TfcResources, report.TfcResources...)
		summary.TfcWorkspaces = append(summary.TfcWorkspaces, report.TfcWorkspaces...)
		summary.Repos = append(summary.Repos, report.Repos...)
		summary.Modules = append(summary.Modules, report.Modules...)
	}
	return summary
}

func FilterReport(report *types.InventoryReport, filter ReportFilter) *types.InventoryReport {
	if report == nil {
		return nil
	}
	filtered := types.InventoryReport{}

	// TODO: this is a mess, refactor
	for _, item := range report.EksClusters {
		if isMatch(item.VersionedResource, filter) {
			filtered.EksClusters = append(filtered.EksClusters, item)
		}
	}
	for _, item := range report.RdsClusters {
		if isMatch(item.VersionedResource, filter) {
			filtered.RdsClusters = append(filtered.RdsClusters, item)
		}
	}
	for _, item := range report.Lambdas {
		if isMatch(item.VersionedResource, filter) {
			filtered.Lambdas = append(filtered.Lambdas, item)
		}
	}
	for _, item := range report.HelmReleases {
		if isMatch(item.VersionedResource, filter) {
			filtered.HelmReleases = append(filtered.HelmReleases, item)
		}
	}
	for _, item := range report.MachineImages {
		if isMatch(item.VersionedResource, filter) {
			filtered.MachineImages = append(filtered.MachineImages, item)
		}
	}
	for _, item := range report.TfcResources {
		if isMatch(item.VersionedResource, filter) {
			filtered.TfcResources = append(filtered.TfcResources, item)
		}
	}
	for _, item := range report.TfcWorkspaces {
		if isMatch(item.VersionedResource, filter) {
			filtered.TfcWorkspaces = append(filtered.TfcWorkspaces, item)
		}
	}
	for _, item := range report.Repos {
		if isMatch(item.VersionedResource, filter) {
			filtered.Repos = append(filtered.Repos, item)
		}
	}
	for _, item := range report.Modules {
		if isMatch(item.VersionedResource, filter) {
			filtered.Modules = append(filtered.Modules, item)
		}
	}

	return &filtered
}

func isMatch(item types.VersionedResource, filter ReportFilter) bool {
	if len(filter.ResourceKinds) > 0 {
		found := false
		for _, kind := range filter.ResourceKinds {
			if item.Kind == kind {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(filter.ParentKinds) > 0 || len(filter.ParentIDs) > 0 {
		for _, parent := range item.Parents {
			kindFound := true
			if len(filter.ParentKinds) > 0 {
				kindFound = false
				for _, kind := range filter.ParentKinds {
					if parent.Kind == kind {
						kindFound = true
						break
					}
				}
			}
			idFound := true
			if len(filter.ParentIDs) > 0 {
				idFound = false
				for _, id := range filter.ParentIDs {
					if parent.ID == id {
						idFound = true
						break
					}
				}
			}
			if !idFound && !kindFound {
				return false
			}
		}
	}

	if len(filter.IDs) > 0 {
		found := false
		for _, id := range filter.IDs {
			if item.ID == id {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(filter.Status) > 0 {
		found := false
		for _, status := range filter.Status {
			if item.EOL.Status == status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(filter.Version) > 0 {
		if item.Version != filter.Version {
			return false
		}
	}

	return true
}

func truncate(s string, l int) string {
	if len(s) > l {
		return fmt.Sprintf("%."+strconv.Itoa(l-1)+"s…", s)
	}
	return s
}

func RemainingDays(eolDate time.Time) int {
	diff := time.Until(eolDate)
	if diff < 0 {
		return 0
	}
	return int(diff.Hours() / 24)
}

func CreateFilter(f []string) ReportFilter {
	filter := ReportFilter{}

	for _, kv := range f {
		parts := strings.Split(kv, "=")
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "status":
			statuses := strings.Split(strings.ToUpper(parts[1]), ",")
			for _, status := range statuses {
				filter.Status = append(filter.Status, types.Status(status))
			}
		case "version":
			filter.Version = parts[1]
		case "id":
			filter.IDs = append(filter.IDs, parts[1])
		case "kind":
			filter.ResourceKinds = append(filter.ResourceKinds, types.ResourceKind(parts[1]))
		case "parent.kind":
			filter.ParentKinds = append(filter.ParentKinds, types.ResourceKind(parts[1]))
		case "parent.id":
			filter.ParentIDs = append(filter.ParentIDs, parts[1])
		}
	}

	return filter
}
