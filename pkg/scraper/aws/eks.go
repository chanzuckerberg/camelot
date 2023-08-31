package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/chanzuckerberg/camelot/pkg/scraper/interfaces"
	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
	"github.com/chanzuckerberg/camelot/pkg/util"
	helmClient "github.com/mittwald/go-helm-client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func extractEksClusterInfo(ctx context.Context, awsClient interfaces.AWSClient) (*types.InventoryReport, error) {
	cycles, err := endOfLife("amazon-eks")
	if err != nil {
		return nil, errors.Wrap(err, "unable to get end of life data")
	}

	activeVersion := ""
	if len(*cycles) > 0 {
		activeVersion = (*cycles)[0].Cycle
	}

	cycleMap := map[string]types.ProductCycle{}
	for _, cycle := range *cycles {
		cycleMap[cycle.Cycle] = cycle
	}

	clusters, err := awsClient.GetEKSClusters()
	if err != nil {
		return nil, errors.Wrap(err, "unable to list clusters")
	}

	var wg sync.WaitGroup
	wg.Add(len(clusters))
	reports := make([]*types.InventoryReport, len(clusters))

	for i, cluster := range clusters {
		go func(cluster string, i int) {
			defer wg.Done()
			report, err := processCluster(ctx, awsClient, cluster, cycleMap, activeVersion)
			if err != nil {
				logrus.Debugf("error processing cluster %s: %s", cluster, err.Error())
				return
			}
			reports[i] = report
		}(cluster, i)
	}
	wg.Wait()

	summary := util.CombineReports(reports)
	return &summary, nil
}

func processCluster(ctx context.Context, awsClient interfaces.AWSClient, cluster string, cycleMap map[string]types.ProductCycle, activeVersion string) (*types.InventoryReport, error) {
	clusterInfo, err := awsClient.DescribeEKSCluster(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "unable to describe cluster")
	}

	config, err := awsClient.GetEKSConfig(ctx, clusterInfo)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get k8s config")
	}

	namespaces, err := awsClient.GetEKSNamespaces(ctx, config)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get k8s namespaces")
	}

	helmReleases, err := getHelmReleases(ctx, config, namespaces, cluster)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get helm releases")
	}

	eol := ""
	if cycle, ok := cycleMap[*clusterInfo.Cluster.Version]; ok {
		eol = fmt.Sprintf("%v", cycle.EOL)
	}

	daysDiff := remainingDays(eol)

	logrus.Debugf("eks cluster: %s -> %s: [%d]", *clusterInfo.Cluster.Arn, *clusterInfo.Cluster.Version, daysDiff)
	addons, err := awsClient.ListEKSAddons(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "unable to describe addons")
	}

	eksAddons := []types.EKSClusterAddon{}
	for _, addon := range addons.Addons {
		addonInfo, err := awsClient.DescribeEKSClusterAddon(cluster, addon)
		if err != nil {
			logrus.Errorf("unable to describe addon: %s", err.Error())
			continue
		}
		logrus.Debugf("    addon: %s -> %s (%s)", addon, *addonInfo.Addon.AddonVersion, addonInfo.Addon.Status)
		eksAddons = append(eksAddons, types.EKSClusterAddon{
			Name:    addon,
			Version: *addonInfo.Addon.AddonVersion,
			Status:  string(addonInfo.Addon.Status),
		})
	}

	eksClusters := []types.EKSCluster{{
		VersionedResource: types.VersionedResource{
			ID:             *clusterInfo.Cluster.Name,
			Kind:           types.KindEKSCluster,
			Arn:            *clusterInfo.Cluster.Arn,
			Parents:        []types.ParentResource{{Kind: types.KindAWSAccount, ID: awsClient.GetAccountId()}},
			Version:        *clusterInfo.Cluster.Version,
			CurrentVersion: activeVersion,
			EOL: types.EOLStatus{
				EOLDate:       eol,
				RemainingDays: daysDiff,
				Status:        eolStatus(daysDiff),
			},
		},
		PlatformVersion: *clusterInfo.Cluster.PlatformVersion,
		Addons:          eksAddons,
	}}

	return &types.InventoryReport{EksClusters: eksClusters, HelmReleases: helmReleases}, nil
}

func createK8sConfig(ctx context.Context, awsClient interfaces.AWSClient, clusterInfo *eks.DescribeClusterOutput, clusterName string) (*rest.Config, error) {
	var rawConfig *rest.Config
	cert, err := base64.StdEncoding.DecodeString(*clusterInfo.Cluster.CertificateAuthority.Data)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode CA data")
	}
	config := api.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: map[string]*api.Cluster{
			"cluster": {
				Server:                   *clusterInfo.Cluster.Endpoint,
				CertificateAuthorityData: cert,
			},
		},
		Contexts: map[string]*api.Context{
			"cluster": {
				Cluster: "cluster",
			},
		},
		CurrentContext: "cluster",
	}
	rawConfig, err = clientcmd.NewDefaultClientConfig(config, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, errors.Wrap(err, "unable to create kubeconfig")
	}
	sc := sts.NewFromConfig(awsClient.GetConfig())
	stsClient := sts.NewPresignClient(sc)

	presignedURLRequest, _ := stsClient.PresignGetCallerIdentity(ctx, &sts.GetCallerIdentityInput{}, func(presignOptions *sts.PresignOptions) {
		presignOptions.ClientOptions = append(presignOptions.ClientOptions, func(stsOptions *sts.Options) {
			stsOptions.APIOptions = append(stsOptions.APIOptions, smithyhttp.SetHeaderValue("x-k8s-aws-id", clusterName))
			stsOptions.APIOptions = append(stsOptions.APIOptions, smithyhttp.SetHeaderValue("X-Amz-Expires", "90"))
		})
	})
	rawConfig.BearerToken = "k8s-aws-v1." + base64.RawURLEncoding.EncodeToString([]byte(presignedURLRequest.URL))

	return rawConfig, nil
}

func getK8sClient(config *rest.Config) (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(config)
}

func getHelmClient(config *rest.Config, namespace string) (helmClient.Client, error) {
	hcClient, err := helmClient.NewClientFromRestConf(&helmClient.RestConfClientOptions{
		Options: &helmClient.Options{
			Namespace: namespace,
			DebugLog:  nil,
		},
		RestConfig: config,
	})
	return hcClient, err
}
