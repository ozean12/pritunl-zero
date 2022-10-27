package ssh

import (
	"fmt"
	"time"

	"github.com/dropbox/godropbox/container/set"
	"github.com/ozean12/pritunl-zero/agent"
	"github.com/ozean12/pritunl-zero/authority"
	"github.com/ozean12/pritunl-zero/database"
	"github.com/ozean12/pritunl-zero/settings"
	"github.com/ozean12/pritunl-zero/user"
	"github.com/ozean12/pritunl-zero/utils"
	"github.com/pritunl/mongo-go-driver/bson"
	"github.com/pritunl/mongo-go-driver/bson/primitive"
	"github.com/pritunl/mongo-go-driver/mongo/options"
)

type Info struct {
	Serial     string    `bson:"serial" json:"serial"`
	Expires    time.Time `bson:"expires" json:"expires"`
	Principals []string  `bson:"principals" json:"principals"`
	Extensions []string  `bson:"extensions" json:"extensions"`
}

type Host struct {
	Domain                string   `bson:"domain" json:"domain"`
	Matches               []string `bson:"matches" json:"matches"`
	ProxyHost             string   `bson:"proxy_host" json:"proxy_host"`
	StrictHostChecking    bool     `bson:"strict_host_checking" json:"strict_host_checking"`
	StrictBastionChecking bool     `bson:"strict_bastion_checking" json:"strict_bastion_checking"`
}

type Certificate struct {
	Id                     primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	UserId                 primitive.ObjectID   `bson:"user_id,omitempty" json:"user_id"`
	AuthorityIds           []primitive.ObjectID `bson:"authority_ids" json:"authority_ids"`
	Timestamp              time.Time            `bson:"timestamp" json:"timestamp"`
	PubKey                 string               `bson:"pub_key"`
	Hosts                  []*Host              `bson:"hosts" json:"hosts"`
	CertificateAuthorities []string             `bson:"certificate_authorities" json:"-"`
	Certificates           []string             `bson:"certificates" json:"-"`
	CertificatesInfo       []*Info              `bson:"certificates_info" json:"certificates_info"`
	Agent                  *agent.Agent         `bson:"agent" json:"agent"`
}

func (c *Certificate) Commit(db *database.Database) (err error) {
	coll := db.SshCertificates()

	err = coll.Commit(c.Id, c)
	if err != nil {
		return
	}

	return
}

func (c *Certificate) CommitFields(db *database.Database, fields set.Set) (
	err error) {

	coll := db.SshCertificates()

	err = coll.CommitFields(c.Id, c, fields)
	if err != nil {
		return
	}

	return
}

func (c *Certificate) Insert(db *database.Database) (err error) {
	coll := db.SshCertificates()

	_, err = coll.InsertOne(db, c)
	if err != nil {
		err = database.ParseError(err)
		return
	}

	return
}

func GetCertificate(db *database.Database, certId primitive.ObjectID) (
	cert *Certificate, err error) {

	coll := db.SshCertificates()
	cert = &Certificate{}

	err = coll.FindOneId(certId, cert)
	if err != nil {
		return
	}

	return
}

func GetCertificates(db *database.Database, userId primitive.ObjectID,
	page, pageCount int64) (certs []*Certificate, count int64, err error) {

	coll := db.SshCertificates()
	certs = []*Certificate{}

	count, err = coll.CountDocuments(db, &bson.M{
		"user_id": userId,
	})
	if err != nil {
		err = database.ParseError(err)
		return
	}

	opts := options.FindOptions{
		Sort: &bson.D{
			{"timestamp", -1},
		},
	}

	if pageCount != 0 {
		maxPage := count / pageCount
		if count == pageCount {
			maxPage = 0
		}
		page = utils.Min64(page, maxPage)
		skip := utils.Min64(page*pageCount, count)
		opts.Skip = &skip
		opts.Limit = &pageCount
	}

	cursor, err := coll.Find(db, &bson.M{
		"user_id": userId,
	}, &opts)
	if err != nil {
		err = database.ParseError(err)
		return
	}
	defer cursor.Close(db)

	for cursor.Next(db) {
		cert := &Certificate{}
		err = cursor.Decode(cert)
		if err != nil {
			err = database.ParseError(err)
			return
		}

		certs = append(certs, cert)
	}

	err = cursor.Err()
	if err != nil {
		err = database.ParseError(err)
		return
	}

	return
}

func NewCertificate(db *database.Database, authrs []*authority.Authority,
	usr *user.User, agnt *agent.Agent, pubKey string) (cert *Certificate,
	err error) {

	cert = &Certificate{
		Id:                     primitive.NewObjectID(),
		UserId:                 usr.Id,
		AuthorityIds:           []primitive.ObjectID{},
		Timestamp:              time.Now(),
		PubKey:                 pubKey,
		Hosts:                  []*Host{},
		CertificateAuthorities: []string{},
		Certificates:           []string{},
		CertificatesInfo:       []*Info{},
		Agent:                  agnt,
	}

	for _, authr := range authrs {
		if !authr.UserHasAccess(usr) {
			continue
		}

		crt, certStr, e := authr.CreateCertificate(db, usr, pubKey)
		if e != nil {
			err = e
			return
		}

		if crt == nil {
			continue
		}

		info := &Info{
			Expires:    time.Unix(int64(crt.ValidBefore), 0),
			Serial:     fmt.Sprintf("%d", crt.Serial),
			Principals: crt.ValidPrincipals,
			Extensions: []string{},
		}

		for permission := range crt.Permissions.Extensions {
			info.Extensions = append(info.Extensions, permission)
		}

		certAuthr := authr.GetCertAuthority()
		if certAuthr != "" {
			cert.CertificateAuthorities = append(
				cert.CertificateAuthorities,
				certAuthr,
			)
		}

		certAuthr = authr.GetBastionCertAuthority()
		if certAuthr != "" {
			cert.CertificateAuthorities = append(
				cert.CertificateAuthorities,
				certAuthr,
			)
		}

		matches, e := authr.GetMatches()
		if e != nil {
			err = e
			return
		}

		if (authr.HostDomain != "" || len(matches) > 0) &&
			(authr.StrictHostChecking || authr.JumpProxy() != "") {

			hst := &Host{
				Domain:             authr.GetHostDomain(),
				ProxyHost:          authr.JumpProxy(),
				Matches:            matches,
				StrictHostChecking: authr.StrictHostChecking,
				StrictBastionChecking: authr.ProxyHosting &&
					!settings.System.DisableBastionHostCertificates,
			}
			cert.Hosts = append(cert.Hosts, hst)
		}

		cert.AuthorityIds = append(cert.AuthorityIds, authr.Id)
		cert.Certificates = append(cert.Certificates, certStr)
		cert.CertificatesInfo = append(cert.CertificatesInfo, info)
	}

	return
}
