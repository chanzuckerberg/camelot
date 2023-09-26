package aws

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	acmtypes "github.com/aws/aws-sdk-go-v2/service/acm/types"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rds_types "github.com/aws/aws-sdk-go-v2/service/rds/types"
	mock_interfaces "github.com/chanzuckerberg/camelot/mocks/mock_aws"
	scraper_types "github.com/chanzuckerberg/camelot/pkg/scraper/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestAwsClient(t *testing.T) {
	r := require.New(t)
	ctrl := gomock.NewController(t)

	mockClient := mock_interfaces.NewMockAWSClient(ctrl)
	mockClient.EXPECT().GetAccountId().Return("123456789012").AnyTimes()

	r.Equal("123456789012", mockClient.GetAccountId())
}

func TestListEKSClusters(t *testing.T) {
	r := require.New(t)

	caData, err := generateCert([]string{"localhost", "endpoint.com"}, "endpoint.com")
	r.NoError(err)

	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM([]byte(caData))
	r.True(ok)

	ctrl := gomock.NewController(t)
	mockClient := mock_interfaces.NewMockAWSClient(ctrl)
	mockClient.EXPECT().GetAccountId().Return("123456789012").AnyTimes()
	mockClient.EXPECT().GetConfig().Return(&aws.Config{
		Region: "us-west-2",
	}).AnyTimes()
	mockClient.EXPECT().GetEKSClusters().Return([]string{"cluster1", "cluster2"}, nil).AnyTimes()
	mockClient.EXPECT().DescribeEKSCluster("cluster1").Return(&eks.DescribeClusterOutput{
		Cluster: &types.Cluster{
			Arn:             &[]string{"arn:aws:eks:us-west-2:123456789012:cluster/cluster1"}[0],
			Name:            &[]string{"cluster1"}[0],
			PlatformVersion: &[]string{"eks.1"}[0],
			Version:         &[]string{"1.27"}[0],
			Endpoint:        &[]string{"https://endpoint.com"}[0],
			CertificateAuthority: &types.Certificate{
				Data: &[]string{string(caData)}[0],
			},
		},
	}, nil).AnyTimes()
	mockClient.EXPECT().DescribeEKSCluster("cluster2").Return(&eks.DescribeClusterOutput{
		Cluster: &types.Cluster{
			Arn:             &[]string{"arn:aws:eks:us-west-2:123456789012:cluster/cluster2"}[0],
			Name:            &[]string{"cluster2"}[0],
			PlatformVersion: &[]string{"eks.1"}[0],
			Version:         &[]string{"1.24"}[0],
			Endpoint:        &[]string{"https://endpoint.com"}[0],
			CertificateAuthority: &types.Certificate{
				Data: &[]string{string(caData)}[0],
			},
		},
	}, nil).AnyTimes()

	mockClient.EXPECT().ListEKSAddons("cluster1").Return(&eks.ListAddonsOutput{}, nil).AnyTimes()
	mockClient.EXPECT().ListEKSAddons("cluster2").Return(&eks.ListAddonsOutput{}, nil).AnyTimes()

	mockClient.EXPECT().GetEKSConfig(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	mockClient.EXPECT().GetEKSNamespaces(gomock.Any(), gomock.Any()).Return([]string{}, nil).AnyTimes()

	report, err := extractEksClusterInfo(context.Background(), mockClient)
	r.NoError(err)
	r.NotNil(report)
	r.Equal(2, len((*report).Resources))
	for _, resource := range (*report).Resources {
		switch resource.(type) {
		case scraper_types.EKSCluster:
		default:
			r.Fail("unexpected type")
		}
		cluster := resource.(scraper_types.EKSCluster)
		r.Equal("eks.1", cluster.PlatformVersion)
		r.GreaterOrEqual(cluster.EOL.RemainingDays, 0)
	}
}

func generateCert(san []string, orgName string) ([]byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   orgName,
			Organization: []string{orgName},
		},
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              san,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, priv.Public(), priv)
	if err != nil {
		return nil, err
	}
	certOut := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: derBytes,
	})

	return certOut, nil
}

func TestListRDSClusters(t *testing.T) {
	r := require.New(t)

	caData, err := generateCert([]string{"localhost", "endpoint.com"}, "endpoint.com")
	r.NoError(err)

	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM([]byte(caData))
	r.True(ok)

	ctrl := gomock.NewController(t)
	mockClient := mock_interfaces.NewMockAWSClient(ctrl)
	mockClient.EXPECT().GetAccountId().Return("123456789012").AnyTimes()
	mockClient.EXPECT().GetConfig().Return(&aws.Config{
		Region: "us-west-2",
	}).AnyTimes()

	mockClient.EXPECT().DescribeRDSClusters().Return(&rds.DescribeDBClustersOutput{
		DBClusters: []rds_types.DBCluster{},
	}, nil)

	_, err = mockClient.DescribeRDSClusters()
	r.NoError(err)
}

func TestListLambdas(t *testing.T) {
	r := require.New(t)

	ctrl := gomock.NewController(t)
	mockClient := mock_interfaces.NewMockAWSClient(ctrl)
	mockClient.EXPECT().GetAccountId().Return("123456789012").AnyTimes()
	mockClient.EXPECT().GetConfig().Return(&aws.Config{
		Region: "us-west-2",
	}).AnyTimes()

	mockClient.EXPECT().ListLambdaFunctions().Return(&lambda.ListFunctionsOutput{}, nil)

	_, err := mockClient.ListLambdaFunctions()
	r.NoError(err)
}

func TestListVolumes(t *testing.T) {
	r := require.New(t)

	ctrl := gomock.NewController(t)
	mockClient := mock_interfaces.NewMockAWSClient(ctrl)
	mockClient.EXPECT().GetAccountId().Return("123456789012").AnyTimes()
	mockClient.EXPECT().GetConfig().Return(&aws.Config{
		Region: "us-west-2",
	}).AnyTimes()

	mockClient.EXPECT().ListVolumes().Return([]ec2types.Volume{
		{
			VolumeId: aws.String("vol-1234567890abcdef0"),
		},
	}, nil)

	vols, err := mockClient.ListVolumes()
	r.NoError(err)
	r.NotEmpty(vols)
}

func TestListCertificates(t *testing.T) {
	r := require.New(t)

	ctrl := gomock.NewController(t)
	mockClient := mock_interfaces.NewMockAWSClient(ctrl)
	mockClient.EXPECT().GetAccountId().Return("123456789012").AnyTimes()
	mockClient.EXPECT().GetConfig().Return(&aws.Config{
		Region: "us-west-2",
	}).AnyTimes()

	mockClient.EXPECT().ListACMCertificates().Return([]acmtypes.CertificateSummary{
		{
			CertificateArn: aws.String("arn:aws:acm:us-west-2:123456789012:certificate/12345678-1234-1234-1234-123456789012"),
		},
	}, nil)

	vols, err := mockClient.ListACMCertificates()
	r.NoError(err)
	r.NotEmpty(vols)
}
