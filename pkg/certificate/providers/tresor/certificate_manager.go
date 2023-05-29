package tresor

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"strings"
	"time"

	"github.com/flomesh-io/fsm/pkg/certificate"
	"github.com/flomesh-io/fsm/pkg/certificate/pem"
	"github.com/flomesh-io/fsm/pkg/errcode"
)

// New constructs a new certificate client using a certificate
func New(
	ca *certificate.Certificate,
	certificatesOrganization string,
	keySize int) (*CertManager, error) {
	if ca == nil {
		return nil, errNoIssuingCA
	}

	if keySize == 0 {
		return nil, fmt.Errorf("key bit size cannot be zero")
	}

	certManager := CertManager{
		// The root certificate signing all newly issued certificates
		ca:                       ca,
		certificatesOrganization: certificatesOrganization,
		keySize:                  keySize,
	}
	return &certManager, nil
}

// IssueCertificate requests a new signed certificate from the configured cert-manager issuer.
func (cm *CertManager) IssueCertificate(cn certificate.CommonName, saNames []string, validityPeriod time.Duration) (*certificate.Certificate, error) {
	if cm.ca == nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrInvalidCA)).
			Msgf("Invalid CA provided for issuance of certificate with CN=%s", cn)
		return nil, errNoIssuingCA
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, cm.keySize)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrGeneratingPrivateKey)).
			Msgf("Error generating private key for certificate with CN=%s", cn)
		return nil, fmt.Errorf("%s: %w", errGeneratingPrivateKey.Error(), err)
	}

	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errGeneratingSerialNumber.Error(), err)
	}

	now := time.Now()
	template := x509.Certificate{
		SerialNumber: serialNumber,

		DNSNames: []string{string(cn)},

		Subject: pkix.Name{
			CommonName:   string(cn),
			Organization: []string{cm.certificatesOrganization},
		},
		NotBefore: now,
		NotAfter:  now.Add(validityPeriod),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	if len(saNames) > 0 {
		template.DNSNames = append(template.DNSNames, saNames...)
	}

	if len(template.DNSNames) > 1 {
		template.DNSNames = uniqueSubjectAlternativeNames(template.DNSNames)
	}

	x509Root, err := certificate.DecodePEMCertificate(cm.ca.GetCertificateChain())
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDecodingPEMCert)).
			Msg("Error decoding Root Certificate's PEM")
		return nil, fmt.Errorf("%s: %w", errCreateCert.Error(), err)
	}

	rsaKeyRoot, err := certificate.DecodePEMPrivateKey(cm.ca.GetPrivateKey())
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrDecodingPEMPrivateKey)).
			Msg("Error decoding Root Certificate's Private Key PEM ")
		return nil, fmt.Errorf("%s: %w", errCreateCert.Error(), err)
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, x509Root, &certPrivKey.PublicKey, rsaKeyRoot)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrCreatingCert)).
			Msgf("Error issuing x509.CreateCertificate command for SerialNumber=%s", serialNumber)
		return nil, fmt.Errorf("%s: %w", errCreateCert.Error(), err)
	}

	certPEM, err := certificate.EncodeCertDERtoPEM(derBytes)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrEncodingCertDERtoPEM)).
			Msgf("Error encoding certificate with SerialNumber=%s", serialNumber)
		return nil, err
	}

	privKeyPEM, err := certificate.EncodeKeyDERtoPEM(certPrivKey)
	if err != nil {
		// TODO(#3962): metric might not be scraped before process restart resulting from this error
		log.Error().Err(err).Str(errcode.Kind, errcode.GetErrCodeWithMetric(errcode.ErrEncodingKeyDERtoPEM)).
			Msgf("Error encoding private key for certificate with SerialNumber=%s", serialNumber)
		return nil, err
	}

	cert := &certificate.Certificate{
		CommonName:   cn,
		SANames:      template.DNSNames,
		SerialNumber: certificate.SerialNumber(serialNumber.String()),
		CertChain:    certPEM,
		PrivateKey:   privKeyPEM,
		IssuingCA:    pem.RootCertificate(cm.ca.GetCertificateChain()),
		TrustedCAs:   pem.RootCertificate(cm.ca.GetCertificateChain()),
		Expiration:   template.NotAfter,
	}

	log.Trace().Msgf("Created new certificate for SerialNumber=%s; validity=%+v; expires on %+v; serial: %x", serialNumber, validityPeriod, template.NotAfter, template.SerialNumber)

	return cert, nil
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
