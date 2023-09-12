package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/chanzuckerberg/camelot/pkg/scraper/interfaces"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func NewAWSClient(ctx context.Context, profile, region, roleARN string) (interfaces.AWSClient, error) {
	client := &awsClient{
		ctx:     ctx,
		profile: profile,
		region:  region,
		roleARN: roleARN,
	}
	err := client.loadConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load config")
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

func (a *awsClient) GetConfig() *aws.Config {
	return a.cfg
}

func (a *awsClient) getAccountId() (string, error) {
	client := sts.NewFromConfig(*a.cfg)
	input := &sts.GetCallerIdentityInput{}

	req, err := client.GetCallerIdentity(a.ctx, input)
	if err != nil {
		return "", errors.Wrap(err, "failed to get caller identity")
	}
	return *req.Account, nil
}

func (a *awsClient) loadConfig() error {
	cfg, err := getAwsConfig(a.ctx, a.profile, a.region, a.roleARN)
	if err != nil {
		return errors.Wrap(err, "failed to load config")
	}
	a.cfg = cfg
	accountId, err := a.getAccountId()
	if err != nil {
		return errors.Wrap(err, "failed to determine identity")
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
		return nil, errors.Wrapf(err, "unable to describe cluster %s", cluster)
	}
	return out, nil
}

func (a *awsClient) ListEKSAddons(cluster string) (*eks.ListAddonsOutput, error) {
	client := eks.NewFromConfig(*a.cfg)
	addons, err := client.ListAddons(a.ctx, &eks.ListAddonsInput{
		ClusterName: &cluster,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to list cluster %s addons", cluster)
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
		return nil, errors.Wrapf(err, "unable to describe cluster %s addon %s", cluster, addon)
	}
	return addonInfo, nil
}

func (a *awsClient) GetEKSConfig(ctx context.Context, clusterInfo *eks.DescribeClusterOutput) (*rest.Config, error) {
	config, err := createK8sConfig(ctx, a, clusterInfo, *clusterInfo.Cluster.Name)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create k8s config for cluster")
	}
	return config, nil
}

func (a *awsClient) GetEKSNamespaces(ctx context.Context, config *rest.Config) ([]string, error) {
	k8sClient, err := getK8sClient(config)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create k8s client for cluster")
	}
	ns, err := k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "unable to list namespaces")
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
		return nil, errors.Wrap(err, "unable to list functions")
	}
	return out, nil
}

func (a *awsClient) DescribeRDSClusters() (*rds.DescribeDBClustersOutput, error) {
	client := rds.NewFromConfig(*a.cfg)

	out, err := client.DescribeDBClusters(a.ctx, &rds.DescribeDBClustersInput{})
	if err != nil {
		return nil, errors.Wrap(err, "unable to list rds clusters")
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

func (a *awsClient) DescribeAMIs(imageIds []string) ([]types.Image, error) {
	client := ec2.NewFromConfig(*a.cfg)

	out, err := client.DescribeImages(a.ctx, &ec2.DescribeImagesInput{
		ImageIds: imageIds,
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to describe AMIs")
	}
	return out.Images, nil
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
		return nil, errors.Wrap(err, "failed to load config")
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
