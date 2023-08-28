package printer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
	"github.com/chanzuckerberg/camelot/pkg/util"
	"github.com/kataras/tablewriter"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func PrintReport(report *types.InventoryReport, outputFormat string) error {
	switch outputFormat {
	case "json":
		b, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return errors.Wrap(err, "failed to marshal json report")
		}
		writer := bufio.NewWriter(os.Stdout)
		_, err = writer.WriteString(string(b))
		if err != nil {
			return errors.Wrap(err, "failed to write yaml report")
		}
		writer.Flush()
	case "yaml":
		b, err := yaml.Marshal(report)
		if err != nil {
			return errors.Wrap(err, "failed to marshal yaml report")
		}
		writer := bufio.NewWriter(os.Stdout)
		_, err = writer.WriteString(string(b))
		if err != nil {
			return errors.Wrap(err, "failed to write yaml report")
		}
		writer.Flush()
	default:
		if len(report.Identity.AwsAccountNumber) > 0 {
			writer := bufio.NewWriter(os.Stdout)
			_, err := writer.WriteString(fmt.Sprintf("\n\nProfile: %s, Account: %s\n\n", report.Identity.AwsProfile, report.Identity.AwsAccountNumber))
			if err != nil {
				return errors.Wrap(err, "failed to write yaml report")
			}
			writer.Flush()
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Kind", "Name", "Parent", "Version", "Current", "Status", "EOL Date"})
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetBorder(false)
		table.SetHeaderLine(false)
		table.SetColumnSeparator("")
		table.SetCenterSeparator("")
		table.AppendBulk(util.ReportToTable(*report))
		table.Render()
	}
	return nil
}
