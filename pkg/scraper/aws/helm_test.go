package aws

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLookupChart(t *testing.T) {
	r := require.New(t)

	packages, err := findHelmChartsByName("cluster-autoscaler")
	r.NoError(err)
	r.NotEmpty(packages)
	packages = filterHelmChartsByHome(packages, "https://kubernetes.github.io/autoscaler")
	r.NotEmpty(packages)
	r.Equal(1, len(packages))

	packages = filterHelmChartsByHome(packages, "https://github.com/kubernetes/autoscaler")
	r.NotEmpty(packages)
	r.Equal(1, len(packages))
}

func TestAliasedUrls(t *testing.T) {
	r := require.New(t)

	r.True(aliasedUrls("https://github.com/kubernetes/autoscaler", "https://kubernetes.github.io/autoscaler"))
	r.True(aliasedUrls("https://github.com/kubernetes-sigs/aws-efs-csi-driver", "https://kubernetes-sigs.github.io/aws-efs-csi-driver/"))
	r.True(aliasedUrls("https://github.com/aws/eks-charts", "https://aws.github.io/eks-charts"))
	r.True(aliasedUrls("https://github.com/kubernetes-sigs/metrics-server/", "https://kubernetes-sigs.github.io/metrics-server"))
}
