package acme

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"strings"

	"github.com/dropbox/godropbox/errors"
	"github.com/ozean12/pritunl-zero/database"
	"github.com/ozean12/pritunl-zero/errortypes"
	"golang.org/x/crypto/acme"
)

func prompt(_ string) bool {
	return true
}

func revoke(client *acme.Client, authzUrls []string) {
	if authzUrls == nil {
		return
	}

	for _, authzUrl := range authzUrls {
		authz, err := client.GetAuthorization(
			context.Background(), authzUrl)
		if err != nil {
			continue
		}

		if authz.Status != acme.StatusPending {
			continue
		}

		_ = client.RevokeAuthorization(context.Background(), authzUrl)
	}
}

func ParsePath(path string) string {
	split := strings.SplitN(path, AcmePath, 2)
	if len(split) == 2 {
		return split[1]
	}
	return ""
}

func GetChallenge(token string) (challenge *Challenge, err error) {
	db := database.GetDatabase()
	defer db.Close()

	coll := db.AcmeChallenges()
	challenge = &Challenge{}

	err = coll.FindOneId(token, challenge)
	if err != nil {
		return
	}

	return
}

func newRsaCsr(domains []string) (csr []byte, keyPem []byte, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "acme: Failed to generate private key"),
		}
		return
	}

	csrReq := &x509.CertificateRequest{
		SignatureAlgorithm: x509.SHA256WithRSA,
		PublicKeyAlgorithm: x509.RSA,
		PublicKey:          key.Public(),
		Subject: pkix.Name{
			CommonName: domains[0],
		},
		DNSNames: domains,
	}

	csr, err = x509.CreateCertificateRequest(rand.Reader, csrReq, key)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "acme: Failed to create certificate request"),
		}
		return
	}

	certKeyByte := x509.MarshalPKCS1PrivateKey(key)

	certKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: certKeyByte,
	}

	keyPem = pem.EncodeToMemory(certKeyBlock)

	return
}

func newEcCsr(domains []string) (csr []byte, keyPem []byte, err error) {
	key, err := ecdsa.GenerateKey(
		elliptic.P384(),
		rand.Reader,
	)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "acme: Failed to generate private key"),
		}
		return
	}

	csrReq := &x509.CertificateRequest{
		SignatureAlgorithm: x509.ECDSAWithSHA256,
		PublicKeyAlgorithm: x509.ECDSA,
		PublicKey:          key.Public(),
		Subject: pkix.Name{
			CommonName: domains[0],
		},
		DNSNames: domains,
	}

	csr, err = x509.CreateCertificateRequest(rand.Reader, csrReq, key)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "acme: Failed to create certificate request"),
		}
		return
	}

	certKeyByte, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		err = &errortypes.ParseError{
			errors.Wrap(err, "acme: Failed to parse private key"),
		}
		return
	}

	certKeyBlock := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: certKeyByte,
	}

	keyPem = pem.EncodeToMemory(certKeyBlock)

	return
}
