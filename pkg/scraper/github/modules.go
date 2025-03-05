package github

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/pkg/errors"
)

type ModuleRef struct {
	Ref       string
	Timestamp time.Time
}

var moduleBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       "module",
			LabelNames: []string{"name"},
		},
	},
}

func findModules(dir string) ([]string, error) {
	modules := []string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".terraform" || d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".tf" {
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		f, diags := hclsyntax.ParseConfig(b, path, hcl.Pos{Line: 1, Column: 1})
		if diags.HasErrors() {
			return errors.Wrapf(diags.Errs()[0], "failed to parse %s", path)
		}

		content, _, diags := f.Body.PartialContent(moduleBlockSchema)
		if diags.HasErrors() {
			return errors.Wrap(diags.Errs()[0], "terraform code has errors")
		}

		for _, block := range content.Blocks {
			if block.Type != "module" {
				continue
			}
			attrs, diags := block.Body.JustAttributes()
			if diags.HasErrors() {
				return fmt.Errorf("terraform code has errors: %w", diags.Errs()[0])
			}
			sourceAttr, ok := attrs["source"]
			if !ok {
				// Module without a source
				continue
			}

			source, diags := sourceAttr.Expr.(*hclsyntax.TemplateExpr).Parts[0].Value(nil)
			if diags.HasErrors() {
				return fmt.Errorf("terraform code has errors: %w", diags.Errs()[0])
			}
			modules = append(modules, source.AsString())
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}
	return modules, nil
}
