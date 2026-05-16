package auth

import "net/http"

const (
	HeaderUserID = "X-Continuum-User-Id"
	HeaderRole   = "X-Continuum-User-Role"
)

type Identity struct {
	UserID  string
	IsAdmin bool
}

func FromRequest(r *http.Request) (Identity, bool) {
	uid := r.Header.Get(HeaderUserID)
	if uid == "" {
		return Identity{}, false
	}
	return Identity{UserID: uid, IsAdmin: r.Header.Get(HeaderRole) == "admin"}, true
}
