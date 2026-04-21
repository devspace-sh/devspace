package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devspace/pkg/devspace/config/constants"

	"github.com/mitchellh/go-homedir"
)

const (
	authCookieName  = "devspace-ui-session"
	authHeaderName  = "X-DevSpace-UI-Token"
	authQueryParam  = "devspace-ui-token"
	authTokenFolder = "sessions"
)

func generateAuthToken() (string, error) {
	token := make([]byte, 32)
	_, err := rand.Read(token)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(token), nil
}

func authMiddleware(h *handler, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.isAuthorized(r) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *handler) index(w http.ResponseWriter, r *http.Request) {
	if !h.authorizeIndexRequest(w, r) {
		return
	}

	http.ServeFile(w, r, filepath.Join(h.path, "index.html"))
}

func (h *handler) authorizeIndexRequest(w http.ResponseWriter, r *http.Request) bool {
	if !h.protectUI {
		return true
	}

	if token := r.URL.Query().Get(authQueryParam); token != "" {
		if !tokensEqual(token, h.authToken) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return false
		}

		h.setAuthCookie(w)

		redirectURL := *r.URL
		query := redirectURL.Query()
		query.Del(authQueryParam)
		redirectURL.RawQuery = query.Encode()
		if redirectURL.Path == "" {
			redirectURL.Path = "/"
		}

		http.Redirect(w, r, redirectURL.String(), http.StatusTemporaryRedirect)
		return false
	}

	if h.isAuthorized(r) {
		return true
	}

	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	return false
}

func (h *handler) setAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     authCookieName,
		Value:    h.authToken,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *handler) isAuthorized(r *http.Request) bool {
	if cookie, err := r.Cookie(authCookieName); err == nil && tokensEqual(cookie.Value, h.authToken) {
		return true
	}

	if token := r.Header.Get(authHeaderName); tokensEqual(token, h.authToken) {
		return true
	}

	authorization := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(authorization), "bearer ") && tokensEqual(strings.TrimSpace(authorization[7:]), h.authToken) {
		return true
	}

	return tokensEqual(r.URL.Query().Get(authQueryParam), h.authToken)
}

func tokensEqual(left, right string) bool {
	if left == "" || right == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func (s *Server) BrowserURL() string {
	return browserURL(s.Server.Addr, s.authToken)
}

func BrowserURL(addr string) string {
	token, err := readAuthToken(addr)
	if err != nil {
		return browserURL(addr, "")
	}

	return browserURL(addr, token)
}

func browserURL(addr, token string) string {
	u := &url.URL{
		Scheme: "http",
		Host:   addr,
		Path:   "/",
	}
	if token != "" {
		query := u.Query()
		query.Set(authQueryParam, token)
		u.RawQuery = query.Encode()
	}

	return u.String()
}

func persistAuthToken(addr, token string) error {
	path, err := authTokenPath(addr)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(path), 0700)
	if err != nil {
		return err
	}

	return os.WriteFile(path, []byte(token), 0600)
}

func readAuthToken(addr string) (string, error) {
	path, err := authTokenPath(addr)
	if err != nil {
		return "", err
	}

	token, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(token)), nil
}

func removeAuthToken(addr string) error {
	path, err := authTokenPath(addr)
	if err != nil {
		return err
	}

	err = os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func authTokenPath(addr string) (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	replacer := strings.NewReplacer(":", "_", "/", "_", "\\", "_")
	return filepath.Join(home, constants.DefaultHomeDevSpaceFolder, UITempFolder, authTokenFolder, replacer.Replace(addr)+".token"), nil
}
