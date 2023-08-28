package aws

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	r := require.New(t)

	products := []string{"amazon-eks", "amazon-rds-postgresql", "amazon-rds-mysql", "nodejs", "go", "ruby", "python"}
	for _, product := range products {
		cycles, err := endOfLife(product)
		r.NoError(err, "failed to get end of life for %s", product)
		r.NotEmpty(cycles)
	}
}
