package certificate

import (
	"fmt"
	"strings"
	"time"
)

// IssueOption is an option that can be passed to IssueCertificate.
type IssueOption func(*issueOptions)

type issueOptions struct {
	fullCNProvided   bool
	validityDuration *time.Duration
	saNames          []string
}

func (o *issueOptions) formatCN(prefix, trustDomain string) CommonName {
	if o.fullCNProvided {
		return CommonName(prefix)
	}
	return CommonName(fmt.Sprintf("%s.%s", prefix, trustDomain))
}

func (o *issueOptions) subjectAlternativeNames() []string {
	if len(o.saNames) > 1 {
		o.saNames = uniqueSubjectAlternativeNames(o.saNames)
	}
	return o.saNames
}

func (o *issueOptions) validityPeriod(validityDuration time.Duration) time.Duration {
	if o.validityDuration != nil {
		return *o.validityDuration
	}
	return validityDuration
}

func uniqueSubjectAlternativeNames(saNames []string, excludeSANS ...string) []string {
	if len(saNames) > 1 {
		sanMap := make(map[string]uint8)
		uniqueSans := make([]string, 0)
		for _, san := range saNames {
			if strings.Contains(san, ":") {
				continue
			}
			if len(excludeSANS) > 0 {
				exclude := false
				for _, exs := range excludeSANS {
					if san == exs {
						exclude = true
						break
					}
				}
				if exclude {
					continue
				}
			}
			if _, ok := sanMap[san]; !ok {
				sanMap[san] = 0
				uniqueSans = append(uniqueSans, san)
			}
		}
		return uniqueSans
	}
	return saNames
}

// FullCNProvided tells IssueCertificate that the provided prefix is actually the full trust domain, and not to append
// the issuer's trust domain.
func FullCNProvided() IssueOption {
	return func(opts *issueOptions) {
		opts.fullCNProvided = true
	}
}

// ValidityDurationProvided tells IssueCertificate that the certificate's validity duration.
func ValidityDurationProvided(validityDuration *time.Duration) IssueOption {
	return func(opts *issueOptions) {
		opts.validityDuration = validityDuration
	}
}

// SubjectAlternativeNames tells IssueCertificate that the certificate's subject alternative names.
func SubjectAlternativeNames(saNames ...string) IssueOption {
	return func(opts *issueOptions) {
		opts.saNames = saNames
	}
}
