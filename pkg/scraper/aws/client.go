package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	acmtypes "github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/chanzuckerberg/camelot/pkg/scraper/interfaces"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

type AWSClientOpt func(*awsClient)

func WithRegion(region string) AWSClientOpt {
	return func(c *awsClient) {
		c.region = region
	}
}

func WithProfile(profile string) AWSClientOpt {
	return func(c *awsClient) {
		c.profile = profile
	}
}

func WithRoleARN(roleARN string) AWSClientOpt {
	return func(c *awsClient) {
		c.roleARN = roleARN
	}
}

func NewAWSClient(ctx context.Context, opts ...AWSClientOpt) (interfaces.AWSClient, error) {
	client := &awsClient{
		ctx: ctx,
	}
	for _, opt := range opts {
		opt(client)
	}

	err := client.loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return client, nil
}

type awsClient struct {
	ctx       context.Context
	profile   string
	region    string
	roleARN   string
	cfg       *aws.Config
	accountId string
}

func (a *awsClient) GetAccountId() string {
	return a.accountId
}

func (a *awsClient) GetProfile() string {
	return a.profile
}

func (a *awsClient) GetConfig() *aws.Config {
	return a.cfg
}

func (a *awsClient) getAccountId() (string, error) {
	client := sts.NewFromConfig(*a.cfg)
	input := &sts.GetCallerIdentityInput{}

	req, err := client.GetCallerIdentity(a.ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get caller identity: %w", err)
	}
	return *req.Account, nil
}

func (a *awsClient) loadConfig() error {
	cfg, err := getAwsConfig(a.ctx, a.profile, a.region, a.roleARN)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	a.cfg = cfg
	accountId, err := a.getAccountId()
	if err != nil {
		return fmt.Errorf("failed to determine identity: %w", err)
	}
	a.accountId = accountId
	return nil
}

func (a *awsClient) GetEKSClusters() ([]string, error) {
	client := eks.NewFromConfig(*a.cfg)
	out, err := client.ListClusters(a.ctx, &eks.ListClustersInput{})
	if err != nil {
		logrus.Errorf("unable to list clusters: %s", err.Error())
		return nil, err
	}
	return out.Clusters, nil
}

func (a *awsClient) DescribeEKSCluster(cluster string) (*eks.DescribeClusterOutput, error) {
	client := eks.NewFromConfig(*a.cfg)
	out, err := client.DescribeCluster(a.ctx, &eks.DescribeClusterInput{
		Name: &cluster,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to describe cluster %s", cluster)
	}
	return out, nil
}

func (a *awsClient) ListEKSAddons(cluster string) (*eks.ListAddonsOutput, error) {
	client := eks.NewFromConfig(*a.cfg)
	addons, err := client.ListAddons(a.ctx, &eks.ListAddonsInput{
		ClusterName: &cluster,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list cluster %s addons", cluster)
	}
	return addons, nil
}

func (a *awsClient) DescribeEKSClusterAddon(cluster, addon string) (*eks.DescribeAddonOutput, error) {
	client := eks.NewFromConfig(*a.cfg)
	addonInfo, err := client.DescribeAddon(a.ctx, &eks.DescribeAddonInput{
		ClusterName: &cluster,
		AddonName:   &addon,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to describe cluster %s addon %s", cluster, addon)
	}
	return addonInfo, nil
}

func (a *awsClient) GetEKSConfig(ctx context.Context, clusterInfo *eks.DescribeClusterOutput) (*rest.Config, error) {
	config, err := createK8sConfig(ctx, a, clusterInfo, *clusterInfo.Cluster.Name)
	if err != nil {
		return nil, fmt.Errorf("unable to create k8s config for cluster")
	}
	return config, nil
}

func (a *awsClient) GetEKSNamespaces(ctx context.Context, config *rest.Config) ([]string, error) {
	k8sClient, err := getK8sClient(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create k8s client for cluster")
	}
	ns, err := k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to list namespaces")
	}
	namespaces := []string{}
	for _, namespace := range ns.Items {
		namespaces = append(namespaces, namespace.Name)
	}
	return namespaces, nil
}

func (a *awsClient) ListLambdaFunctions() (*lambda.ListFunctionsOutput, error) {
	client := lambda.NewFromConfig(*a.cfg)
	out, err := client.ListFunctions(a.ctx, &lambda.ListFunctionsInput{})
	if err != nil {
		return nil, fmt.Errorf("unable to list functions")
	}
	return out, nil
}

func (a *awsClient) DescribeRDSClusters() (*rds.DescribeDBClustersOutput, error) {
	client := rds.NewFromConfig(*a.cfg)

	out, err := client.DescribeDBClusters(a.ctx, &rds.DescribeDBClustersInput{})
	if err != nil {
		return nil, fmt.Errorf("unable to list rds clusters")
	}
	return out, nil
}

func (a *awsClient) ListEC2Instances() ([]types.Instance, error) {
	instances := []types.Instance{}
	client := ec2.NewFromConfig(*a.cfg)
	var token *string
	for {
		out, err := client.DescribeInstances(a.ctx, &ec2.DescribeInstancesInput{
			NextToken: token,
			Filters: []types.Filter{
				{
					Name: aws.String("instance-state-name"),
					Values: []string{
						"running",
					},
				},
			},
		})
		if err != nil {
			logrus.Errorf("unable to list ec2 instances: %s", err.Error())
			break
		}
		for _, reservation := range out.Reservations {
			instances = append(instances, reservation.Instances...)
		}
		if out.NextToken == nil {
			break
		}
	}
	return instances, nil
}

func (a *awsClient) ListVolumes() ([]types.Volume, error) {
	volumes := []types.Volume{}
	client := ec2.NewFromConfig(*a.cfg)

	var token *string
	for {
		out, err := client.DescribeVolumes(a.ctx, &ec2.DescribeVolumesInput{
			NextToken: token,
		})

		if err != nil {
			logrus.Errorf("unable to list volumes: %s", err.Error())
			break
		}

		volumes = append(volumes, out.Volumes...)

		if out.NextToken == nil {
			break
		}
	}
	return volumes, nil
}

func (a *awsClient) DescribeAMIs(imageIds []string) ([]types.Image, error) {
	images := []types.Image{}
	client := ec2.NewFromConfig(*a.cfg)

	var token *string
	for {
		out, err := client.DescribeImages(a.ctx, &ec2.DescribeImagesInput{ImageIds: imageIds, NextToken: token})

		if err != nil {
			logrus.Errorf("unable to list AMIs: %s", err.Error())
			break
		}

		images = append(images, out.Images...)

		if out.NextToken == nil {
			break
		}
	}

	return images, nil
}

func (a *awsClient) ListACMCertificates() ([]acmtypes.CertificateSummary, error) {
	certificates := []acmtypes.CertificateSummary{}
	client := acm.NewFromConfig(*a.cfg)

	var token *string
	for {
		out, err := client.ListCertificates(a.ctx, &acm.ListCertificatesInput{NextToken: token})

		if err != nil {
			logrus.Errorf("unable to list certificates: %s", err.Error())
			break
		}

		certificates = append(certificates, out.CertificateSummaryList...)

		if out.NextToken == nil {
			break
		}
	}
	return certificates, nil
}

func getAwsConfig(ctx context.Context, profile, region, roleARN string) (*aws.Config, error) {
	opts := []func(*config.LoadOptions) error{}
	if len(profile) > 0 {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}
	if len(region) > 0 {
		opts = append(opts, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load config")
	}

	if len(roleARN) > 0 {
		stsClient := sts.NewFromConfig(cfg)
		roleCreds := stscreds.NewAssumeRoleProvider(stsClient, roleARN)
		roleCfg := cfg.Copy()
		roleCfg.Credentials = aws.NewCredentialsCache(roleCreds)
		cfg = roleCfg
	}
	return &cfg, nil
}
