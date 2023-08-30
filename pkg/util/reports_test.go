package util

import (
	"testing"

	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
	"github.com/stretchr/testify/require"
)

func TestCreateFilter(t *testing.T) {
	r := require.New(t)
	f := CreateFilter([]string{"kind=" + string(types.KindEKSCluster)})
	r.True(len(f.ResourceKinds) == 1)
	r.True(f.ResourceKinds[0] == types.KindEKSCluster)

	f = CreateFilter([]string{"parent.kind=" + string(types.KindEKSCluster)})
	r.True(len(f.ParentKinds) == 1)
	r.True(f.ParentKinds[0] == types.KindEKSCluster)

	f = CreateFilter([]string{"parent.kind=" + string(types.KindAWSAccount), "parent.id=123456789012"})
	r.True(len(f.ParentKinds) == 1)
	r.True(len(f.ParentIDs) == 1)
	r.True(len(f.IDs) == 0)
	r.True(f.ParentKinds[0] == types.KindAWSAccount)
	r.True(f.ParentIDs[0] == "123456789012")

	f = CreateFilter([]string{"parent.kind=" + string(types.KindAWSAccount), "parent.id=123456789012", "id=abc"})
	r.True(len(f.ParentKinds) == 1)
	r.True(len(f.ParentIDs) == 1)
	r.True(len(f.IDs) == 1)
	r.True(f.ParentKinds[0] == types.KindAWSAccount)
	r.True(f.ParentIDs[0] == "123456789012")
	r.True(f.IDs[0] == "abc")

	f = CreateFilter([]string{"status=active"})
	r.True(f.Status == types.StatusActive)
}
