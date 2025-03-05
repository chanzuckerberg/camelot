package github

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
)

var constraintRegexp *regexp.Regexp

func init() {
	// Liberally borrowed from hashicorp/go-version to avoid using reflection
	constraintOperators := []string{"", "=", "!=", ">", "<", ">=", "<=", "~>"}

	ops := make([]string, 0, len(constraintOperators))
	for _, op := range constraintOperators {
		ops = append(ops, regexp.QuoteMeta(op))
	}

	constraintRegexp = regexp.MustCompile(fmt.Sprintf(
		`^\s*(%s)\s*(%s)\s*$`,
		strings.Join(ops, "|"),
		version.VersionRegexpRaw))
}

func findOldestVersionConstraint(c string) (*version.Version, error) {
	parts := strings.Split(c, ",")
	versions := []*version.Version{}
	for _, part := range parts {
		matches := constraintRegexp.FindStringSubmatch(part)
		if matches == nil {
			return nil, errors.Errorf("invalid constraint %s", part)
		}
		ver, err := version.NewVersion(matches[2])
		if err != nil {
			return nil, fmt.Errorf("unable to parse version %s", matches[2])
		}
		versions = append(versions, ver)
	}
	if len(versions) == 0 {
		return nil, errors.New("no versions found")
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].LessThan(versions[j])
	})
	return versions[0], nil
}
