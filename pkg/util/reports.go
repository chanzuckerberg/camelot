package util

import (
	"fmt"
	"strconv"
	"time"

	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
)

func ReportToTable(report types.InventoryReport) [][]string {
	var table [][]string
	for _, item := range report.MachineImages {
		table = append(table, versionedResourceToTableRow("AMI", item.VersionedResource))
	}
	for _, item := range report.RdsClusters {
		table = append(table, versionedResourceToTableRow("RDS", item.VersionedResource))
	}
	for _, item := range report.EksClusters {
		table = append(table, versionedResourceToTableRow("EKS", item.VersionedResource))
	}
	for _, item := range report.Lambdas {
		table = append(table, versionedResourceToTableRow("Lambda", item.VersionedResource))
	}
	for _, item := range report.Repos {
		table = append(table, versionedResourceToTableRow("GitRepo", item.VersionedResource))
	}
	for _, item := range report.Modules {
		table = append(table, versionedResourceToTableRow("TfModule", item.VersionedResource))
	}
	for _, item := range report.HelmReleases {
		table = append(table, versionedResourceToTableRow("HelmRelease", item.VersionedResource))
	}
	for _, item := range report.TfcResources {
		table = append(table, versionedResourceToTableRow("TfcResource", item.VersionedResource))
	}
	for _, item := range report.TfcWorkspaces {
		table = append(table, versionedResourceToTableRow("TfcWorkspace", item.VersionedResource))
	}
	return table
}

func versionedResourceToTableRow(kind string, item types.VersionedResource) []string {
	return []string{
		kind,
		truncate(item.Name, 80),
		truncate(item.Parent, 80),
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
	}
	return summary
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
