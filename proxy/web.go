package proxy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"github.com/ozean12/pritunl-zero/authorizer"
	"github.com/ozean12/pritunl-zero/logger"
	"github.com/ozean12/pritunl-zero/node"
	"github.com/ozean12/pritunl-zero/searches"
	"github.com/ozean12/pritunl-zero/service"
	"github.com/ozean12/pritunl-zero/settings"
	"github.com/ozean12/pritunl-zero/utils"
	"github.com/sirupsen/logrus"
)

type web struct {
	reqHost     string
	serverHost  string
	serverProto string
	proxyProto  string
	proxyPort   int
	Transport   http.RoundTripper
	ErrorLog    *log.Logger
}

func (w *web) ServeHTTP(rw http.ResponseWriter, r *http.Request,
	authr *authorizer.Authorizer) {

	prxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.Header.Set("X-Forwarded-For",
				node.Self.GetRemoteAddr(req))
			req.Header.Set("X-Forwarded-Host", req.Host)
			req.Header.Set("X-Forwarded-Proto", w.proxyProto)
			req.Header.Set("X-Forwarded-Port", strconv.Itoa(w.proxyPort))

			if authr != nil {
				usr, _ := authr.GetUser(nil)
				if usr != nil {
					req.Header.Set("X-Forwarded-User", usr.Username)
				}
			}

			if w.reqHost != "" {
				req.Host = w.reqHost
			}

			req.URL.Scheme = w.serverProto
			req.URL.Host = w.serverHost

			stripCookieHeaders(req)

			if settings.Elastic.ProxyRequests {
				index := searches.Request{
					Address:   node.Self.GetRemoteAddr(req),
					Timestamp: time.Now(),
					Scheme:    req.URL.Scheme,
					Host:      req.URL.Host,
					Path:      req.URL.Path,
					Query:     req.URL.Query(),
					Header:    req.Header.Clone(),
				}

				if authr.IsValid() {
					usr, _ := authr.GetUser(nil)

					if usr != nil {
						index.User = usr.Id.Hex()
						index.Username = usr.Username
						index.Session = authr.SessionId()
					}
				}

				contentType := strings.ToLower(req.Header.Get("Content-Type"))
				if searches.RequestTypes.Contains(contentType) &&
					req.ContentLength != 0 &&
					req.Body != nil {

					bodyCopy := &bytes.Buffer{}
					tee := io.TeeReader(req.Body, bodyCopy)
					body, _ := ioutil.ReadAll(tee)
					_ = req.Body.Close()
					req.Body = utils.NopCloser{bodyCopy}
					index.Body = string(body)
				}

				index.Index()
			}
		},
		Transport: w.Transport,
		ErrorLog:  w.ErrorLog,
	}

	prxy.ServeHTTP(rw, r)
}

func newWeb(proxyProto string, proxyPort int, host *Host,
	server *service.Server) (w *web) {

	dialTimeout := time.Duration(
		settings.Router.DialTimeout) * time.Second
	dialKeepAlive := time.Duration(
		settings.Router.DialKeepAlive) * time.Second
	maxIdleConns := settings.Router.MaxIdleConns
	maxIdleConnsPerHost := settings.Router.MaxIdleConnsPerHost
	idleConnTimeout := time.Duration(
		settings.Router.IdleConnTimeout) * time.Second
	handshakeTimeout := time.Duration(
		settings.Router.HandshakeTimeout) * time.Second
	continueTimeout := time.Duration(
		settings.Router.ContinueTimeout) * time.Second
	headerTimeout := time.Duration(
		settings.Router.HeaderTimeout) * time.Second

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
	}
	if settings.Router.SkipVerify || net.ParseIP(server.Hostname) != nil {
		tlsConfig.InsecureSkipVerify = true
	}

	if host.ClientCertificate != nil {
		tlsConfig.Certificates = []tls.Certificate{
			*host.ClientCertificate,
		}
	}

	writer := &logger.ErrorWriter{
		Message: "node: Proxy server error",
		Fields: logrus.Fields{
			"service": host.Service.Name,
			"domain":  host.Domain.Domain,
			"server": fmt.Sprintf(
				"%s://%s:%d",
				server.Protocol,
				server.Hostname,
				server.Port,
			),
		},
		Filters: []string{
			"context canceled",
		},
	}

	w = &web{
		reqHost:     host.Domain.Host,
		serverProto: server.Protocol,
		serverHost:  utils.FormatHostPort(server.Hostname, server.Port),
		proxyProto:  proxyProto,
		proxyPort:   proxyPort,
		Transport: &TransportFix{
			transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   dialTimeout,
					KeepAlive: dialKeepAlive,
					DualStack: true,
				}).DialContext,
				MaxResponseHeaderBytes: int64(
					settings.Router.MaxResponseHeaderBytes),
				MaxIdleConns:          maxIdleConns,
				MaxIdleConnsPerHost:   maxIdleConnsPerHost,
				ResponseHeaderTimeout: headerTimeout,
				IdleConnTimeout:       idleConnTimeout,
				TLSHandshakeTimeout:   handshakeTimeout,
				ExpectContinueTimeout: continueTimeout,
				TLSClientConfig:       tlsConfig,
			},
		},
		ErrorLog: log.New(writer, "", 0),
	}

	return
}
