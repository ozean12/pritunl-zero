package authorizer

import (
	"net/http"

	"github.com/ozean12/pritunl-zero/auth"
	"github.com/ozean12/pritunl-zero/database"
	"github.com/ozean12/pritunl-zero/service"
	"github.com/ozean12/pritunl-zero/signature"
)

func AuthorizeAdmin(db *database.Database, w http.ResponseWriter,
	r *http.Request) (authr *Authorizer, err error) {

	authr = NewAdmin()

	token := r.Header.Get("Pritunl-Zero-Token")
	sigStr := r.Header.Get("Pritunl-Zero-Signature")

	if token != "" && sigStr != "" {
		timestamp := r.Header.Get("Pritunl-Zero-Timestamp")
		nonce := r.Header.Get("Pritunl-Zero-Nonce")

		sig, e := signature.Parse(
			token,
			sigStr,
			timestamp,
			nonce,
			r.Method,
			r.URL.Path,
		)
		if e != nil {
			err = e
			return
		}

		err = authr.AddSignature(db, sig)
		if err != nil {
			return
		}
	} else {
		cook, sess, e := auth.CookieSessionAdmin(db, w, r)
		if e != nil {
			err = e
			return
		}

		err = authr.AddCookie(cook, sess)
		if err != nil {
			return
		}
	}

	return
}

func AuthorizeProxy(db *database.Database, srvc *service.Service,
	w http.ResponseWriter, r *http.Request) (authr *Authorizer, err error) {

	authr = NewProxy(srvc)

	token := r.Header.Get("Pritunl-Zero-Token")
	sigStr := r.Header.Get("Pritunl-Zero-Signature")

	if token != "" && sigStr != "" {
		timestamp := r.Header.Get("Pritunl-Zero-Timestamp")
		nonce := r.Header.Get("Pritunl-Zero-Nonce")

		sig, e := signature.Parse(
			token,
			sigStr,
			timestamp,
			nonce,
			r.Method,
			r.URL.Path,
		)
		if e != nil {
			err = e
			return
		}

		err = authr.AddSignature(db, sig)
		if err != nil {
			return
		}
	} else {
		cook, sess, e := auth.CookieSessionProxy(db, srvc, w, r)
		if e != nil {
			err = e
			return
		}

		err = authr.AddCookie(cook, sess)
		if err != nil {
			return
		}
	}

	return
}

func AuthorizeUser(db *database.Database, w http.ResponseWriter,
	r *http.Request) (authr *Authorizer, err error) {

	authr = NewUser()

	token := r.Header.Get("Pritunl-Zero-Token")
	sigStr := r.Header.Get("Pritunl-Zero-Signature")

	if token != "" && sigStr != "" {
		timestamp := r.Header.Get("Pritunl-Zero-Timestamp")
		nonce := r.Header.Get("Pritunl-Zero-Nonce")

		sig, e := signature.Parse(
			token,
			sigStr,
			timestamp,
			nonce,
			r.Method,
			r.URL.Path,
		)
		if e != nil {
			err = e
			return
		}

		err = authr.AddSignature(db, sig)
		if err != nil {
			return
		}
	} else {
		cook, sess, e := auth.CookieSessionUser(db, w, r)
		if e != nil {
			err = e
			return
		}

		err = authr.AddCookie(cook, sess)
		if err != nil {
			return
		}
	}

	return
}

func NewAdmin() (authr *Authorizer) {
	authr = &Authorizer{
		typ: Admin,
	}

	return
}

func NewProxy(srvc *service.Service) (authr *Authorizer) {
	authr = &Authorizer{
		typ:  Proxy,
		srvc: srvc,
	}

	return
}

func NewUser() (authr *Authorizer) {
	authr = &Authorizer{
		typ: User,
	}

	return
}
