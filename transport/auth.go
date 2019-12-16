package transport

import (
	"net/http"
)

type AuthScheme string

const (
	AuthSchemeClockify = "clockify"
	AuthSchemeToggl    = "toggl"
)

type AuthTransport struct {
	next http.RoundTripper

	scheme AuthScheme
	user   string
	token  string
}

type AuthOptions struct {
	scheme AuthScheme
	User   string
	Token  string
}

func NewAuthTransport(scheme AuthScheme, opts AuthOptions, next http.RoundTripper) http.RoundTripper {
	return &AuthTransport{next, scheme, opts.User, opts.Token}
}

func (a *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch a.scheme {
	case AuthSchemeClockify:
		req.Header.Add("X-Api-Key", a.token)
	case AuthSchemeToggl:
		req.SetBasicAuth(a.user, a.token)
	}
	return a.next.RoundTrip(req)
}
