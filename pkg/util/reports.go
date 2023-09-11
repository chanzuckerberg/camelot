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
	for _, item := range report.Resources {
		table = append(table, versionedResourceToTableRow(item.GetVersionedResource()))
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
		summary.Resources = append(summary.Resources, report.Resources...)
	}
	return summary
}

func FilterReport(report *types.InventoryReport, filter ReportFilter) *types.InventoryReport {
	if report == nil {
		return nil
	}
	filtered := types.InventoryReport{}

	for _, item := range report.Resources {
		if isMatch(item.GetVersionedResource(), filter) {
			filtered.Resources = append(filtered.Resources, item)
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
			if !idFound || !kindFound {
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
