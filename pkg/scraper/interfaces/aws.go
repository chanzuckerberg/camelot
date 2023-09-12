package interfaces

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"k8s.io/client-go/rest"
)

//go:generate mockgen -source=$GOFILE -destination=../../../mocks/mock_aws/mock_$GOFILE
type AWSClient interface {
	GetAccountId() string
	GetConfig() *aws.Config
	GetEKSClusters() ([]string, error)
	DescribeEKSCluster(cluster string) (*eks.DescribeClusterOutput, error)
	ListEKSAddons(cluster string) (*eks.ListAddonsOutput, error)
	DescribeEKSClusterAddon(cluster, addon string) (*eks.DescribeAddonOutput, error)
	ListLambdaFunctions() (*lambda.ListFunctionsOutput, error)
	DescribeRDSClusters() (*rds.DescribeDBClustersOutput, error)
	GetEKSConfig(ctx context.Context, clusterInfo *eks.DescribeClusterOutput) (*rest.Config, error)
	GetEKSNamespaces(ctx context.Context, config *rest.Config) ([]string, error)
	ListEC2Instances() ([]ec2types.Instance, error)
	DescribeAMIs(imageIds []string) ([]ec2types.Image, error)
}
