package cookie

import (
	"net/http"
	"strings"

	"github.com/dropbox/godropbox/errors"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/ozean12/pritunl-zero/errortypes"
	"github.com/ozean12/pritunl-zero/service"
	"github.com/ozean12/pritunl-zero/settings"
	"github.com/ozean12/pritunl-zero/utils"
)

func GetAdmin(w http.ResponseWriter, r *http.Request) (
	cook *Cookie, err error) {

	store, err := Store.New(r, "pritunl-zero-console")
	if err != nil {
		err = &errortypes.UnknownError{
			errors.Wrap(err.(securecookie.MultiError)[0],
				"cookie: Unknown cookie error"),
		}
		return
	}

	cook = &Cookie{
		store: store,
		w:     w,
		r:     r,
	}

	return
}

func NewAdmin(w http.ResponseWriter, r *http.Request) (cook *Cookie) {
	store, _ := Store.New(r, "pritunl-zero-console")

	cook = &Cookie{
		store: store,
		w:     w,
		r:     r,
	}

	return
}

func CleanAdmin(w http.ResponseWriter, r *http.Request) {
	cook := &http.Cookie{
		Name:     "pritunl-zero-console",
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		MaxAge:   -1,
	}
	http.SetCookie(w, cook)

	return
}

func getCookieTopDomain(host string) string {
	host = utils.StripPort(host)

	if host == "" {
		return ""
	}

	minLen := 0
	hostSpl := strings.Split(host, ".")
	if len(hostSpl[len(hostSpl)-1]) == 2 {
		minLen = 3
	} else {
		minLen = 2
	}

	if strings.Count(host, ".") >= minLen {
		host = "." + strings.SplitN(host, ".", 2)[1]
		return host
	}

	return ""
}

func getCookieNextDomain(host string) string {
	host = utils.StripPort(host)

	if host == "" {
		return ""
	}

	if host[0] == '.' {
		host = host[1:]
	}

	minLen := 0
	hostSpl := strings.Split(host, ".")
	if len(hostSpl[len(hostSpl)-1]) == 2 {
		minLen = 3
	} else {
		minLen = 2
	}

	if strings.Count(host, ".") >= minLen {
		host = "." + strings.SplitN(host, ".", 2)[1]
		return host
	}

	return ""
}

func newProxyStore(srvc *service.Service,
	r *http.Request) *sessions.CookieStore {

	cookieStore := sessions.NewCookieStore(
		settings.System.ProxyCookieAuthKey,
		settings.System.ProxyCookieCryptoKey,
	)
	cookieStore.Options.Secure = true
	cookieStore.Options.HttpOnly = true

	if srvc.ShareSession {
		cookieStore.Options.Domain = getCookieTopDomain(r.Host)
	}

	return cookieStore
}

func GetProxy(srvc *service.Service, w http.ResponseWriter, r *http.Request) (
	cook *Cookie, err error) {

	cookStore := newProxyStore(srvc, r)

	store, err := cookStore.New(r, "pritunl-zero")
	if err != nil {
		err = &errortypes.UnknownError{
			errors.Wrap(err.(securecookie.MultiError)[0],
				"cookie: Unknown cookie error"),
		}
		return
	}

	cook = &Cookie{
		store: store,
		w:     w,
		r:     r,
	}

	return
}

func NewProxy(srvc *service.Service, w http.ResponseWriter, r *http.Request) (
	cook *Cookie) {

	cookStore := newProxyStore(srvc, r)

	store, _ := cookStore.New(r, "pritunl-zero")

	cook = &Cookie{
		store: store,
		w:     w,
		r:     r,
	}

	return
}

func CleanProxy(w http.ResponseWriter, r *http.Request) {
	cook := &http.Cookie{
		Name:     "pritunl-zero",
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		MaxAge:   -1,
	}
	http.SetCookie(w, cook)

	domain := getCookieTopDomain(r.Host)
	if domain != "" {
		cook = &http.Cookie{
			Name:     "pritunl-zero",
			Path:     "/",
			Secure:   true,
			HttpOnly: true,
			MaxAge:   -1,
			Domain:   domain,
		}
		http.SetCookie(w, cook)

		for i := 0; i < 10; i++ {
			domain = getCookieNextDomain(domain)
			if domain == "" {
				break
			}

			cook = &http.Cookie{
				Name:     "pritunl-zero",
				Path:     "/",
				Secure:   true,
				HttpOnly: true,
				MaxAge:   -1,
				Domain:   domain,
			}
			http.SetCookie(w, cook)
		}
	}

	return
}

func GetUser(w http.ResponseWriter, r *http.Request) (
	cook *Cookie, err error) {

	store, err := UserStore.New(r, "pritunl-zero-user")
	if err != nil {
		err = &errortypes.UnknownError{
			errors.Wrap(err.(securecookie.MultiError)[0],
				"cookie: Unknown cookie error"),
		}
		return
	}

	cook = &Cookie{
		store: store,
		w:     w,
		r:     r,
	}

	return
}

func NewUser(w http.ResponseWriter, r *http.Request) (cook *Cookie) {
	store, _ := UserStore.New(r, "pritunl-zero-user")

	cook = &Cookie{
		store: store,
		w:     w,
		r:     r,
	}

	return
}

func CleanUser(w http.ResponseWriter, r *http.Request) {
	cook := &http.Cookie{
		Name:     "pritunl-zero-user",
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		MaxAge:   -1,
	}
	http.SetCookie(w, cook)

	return
}
