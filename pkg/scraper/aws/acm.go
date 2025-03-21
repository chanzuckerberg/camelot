package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	acmtypes "github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/chanzuckerberg/camelot/pkg/scraper/interfaces"
	"github.com/chanzuckerberg/camelot/pkg/scraper/types"
)

func extractACMCertificates(ctx context.Context, awsClient interfaces.AWSClient) (*types.InventoryReport, error) {
	out, err := awsClient.ListACMCertificates()
	if err != nil {
		return nil, fmt.Errorf("unable to list certificates: %w", err)
	}
	certificates := []types.Versioned{}
	for _, certificate := range out {
		parts := strings.Split(*certificate.CertificateArn, "/")
		var status types.Status = types.StatusValid

		eol := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
		if certificate.NotAfter != nil {
			eol = certificate.NotAfter.Format("2006-01-02")
		}

		switch certificate.Status {
		case acmtypes.CertificateStatusPendingValidation:
			status = types.StatusWarning
		case acmtypes.CertificateStatusFailed:
			status = types.StatusCritical
		case acmtypes.CertificateStatusRevoked:
			status = types.StatusCritical
			eol = certificate.RevokedAt.Format("2006-01-02")
		case acmtypes.CertificateStatusInactive:
			status = types.StatusWarning
		case acmtypes.CertificateStatusExpired:
			status = types.StatusCritical
		case acmtypes.CertificateStatusValidationTimedOut:
			status = types.StatusCritical
		case acmtypes.CertificateStatusIssued:
			// Imported certificates are not eligible for renewal, this one is expiring in 30 days
			if certificate.RenewalEligibility == acmtypes.RenewalEligibilityIneligible && certificate.NotAfter.Before(time.Now().AddDate(0, 0, 30)) {
				status = types.StatusWarning
			} else if !*certificate.InUse {
				// Certificate is ussued, but not used
				status = types.StatusWarning
			}
		}

		daysDiff := remainingDays(eol)
		certificates = append(certificates, types.ACMCertificate{
			InUse:            *certificate.InUse,
			Status:           string(certificate.Status),
			Expiration:       eol,
			DomainName:       *certificate.DomainName,
			AlternativeNames: certificate.SubjectAlternativeNameSummaries,
			VersionedResource: types.VersionedResource{
				ID:      parts[1],
				Arn:     *certificate.CertificateArn,
				Kind:    types.KindACMCertificate,
				Parents: []types.ParentResource{{Kind: types.KindAWSAccount, ID: awsClient.GetAccountId()}},
				Version: string(certificate.KeyAlgorithm),
				EOL: types.EOLStatus{
					EOLDate:       eol,
					RemainingDays: daysDiff,
					Status:        status,
				},
			},
		})
	}
	return &types.InventoryReport{
		Resources: certificates,
	}, nil
}
