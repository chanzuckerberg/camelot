package aws

import (
	"testing"
	"time"

	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
	"github.com/stretchr/testify/require"
)

func TestRemainingDays(t *testing.T) {
	r := require.New(t)
	eol := time.Now().AddDate(0, 0, 10)
	days := remainingDays(eol.Format("2006-01-02"))
	r.GreaterOrEqual(days, 9)
}

func TestEolStatus(t *testing.T) {
	r := require.New(t)
	r.Equal(types.StatusCritical, string(eolStatus(0)))
	r.Equal(types.StatusCritical, string(eolStatus(15)))
	r.Equal(types.StatusWarning, string(eolStatus(80)))
}
