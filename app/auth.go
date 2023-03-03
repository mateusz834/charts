package app

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/mateusz834/charts/service"
)

func (a *application) isAuthenticated(w http.ResponseWriter, r *http.Request) error {
	type response struct {
		Auth bool `json:"authenticated"`
	}

	_, err := a.authenticate(r)
	if err != nil {
		if errors.As(err, new(service.PublicError)) {
			return sendJSON(w, http.StatusOK, response{Auth: false})
		}
		return err
	}
	return sendJSON(w, http.StatusOK, response{Auth: true})
}

func (a *application) authenticate(r *http.Request) (uint64, error) {
	cookie, err := r.Cookie("__Host-session")
	if err != nil {
		return 0, service.PublicWrapperError{errors.New("missing valid session cookie")}
	}

	githubUserID, err := a.sessionService.IsSessionValid(cookie.Value)
	if err != nil {
		return 0, err
	}

	return githubUserID, nil
}

func (a *application) githubLogin(w http.ResponseWriter, r *http.Request) error {
	csrfBin := make([]byte, 32)
	if _, err := rand.Read(csrfBin); err != nil {
		return err
	}

	csrf := base64.RawURLEncoding.EncodeToString(csrfBin)

	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-oauth-state",
		Value:    csrf,
		Path:     "/",
		MaxAge:   3600,
		Expires:  time.Now().Add(time.Hour),
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
		Secure:   true,
	})

	authUrl, err := url.Parse("https://github.com/login/oauth/authorize")
	if err != nil {
		return err
	}
	query := url.Values{}
	query.Add("response_type", "code")
	query.Add("client_id", "14e6190e978637376f67")
	query.Add("state", csrf)
	authUrl.RawQuery = query.Encode()

	http.Redirect(w, r, authUrl.String(), http.StatusFound)
	return nil
}

func (a *application) githubLoginCallback(w http.ResponseWriter, r *http.Request) error {
	query := r.URL.Query()
	if _, ok := query["error"]; ok {
		http.Redirect(w, r, "/", http.StatusFound)
		return nil
	}

	code := query.Get("code")
	if len(code) == 0 {
		return &httpError{
			DebugErr:     errors.New("missing code query param"),
			ResponseCode: http.StatusBadRequest,
		}
	}

	state := query.Get("state")
	if len(code) == 0 {
		return &httpError{
			DebugErr:     errors.New("missing state query param"),
			ResponseCode: http.StatusBadRequest,
		}
	}

	stateCookie, err := r.Cookie("__Host-oauth-state")
	if err != nil {
		return &httpError{
			DebugErr:     errors.New("missing __Host-oauth-state cookie"),
			ResponseCode: http.StatusBadRequest,
		}
	}

	if state != stateCookie.Value {
		return &httpError{
			DebugErr:     errors.New("login csrf, bad state url query param"),
			ResponseCode: http.StatusBadRequest,
		}
	}

	accessToken, err := a.githubOAuth.getAccessToken(code)
	if err != nil {
		return err
	}

	userData, err := getGithubUserData(accessToken)
	if err != nil {
		return err
	}

	s, err := a.sessionService.NewSession(userData.ID)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "__Host-oauth-state",
		Path:    "/",
		Expires: time.UnixMicro(0),
		MaxAge:  -1,
		Secure:  true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "__Host-session",
		Value:    s,
		Path:     "/",
		MaxAge:   3600 * 24 * 7,
		Expires:  time.Now().Add(time.Hour * 24 * 7),
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
		Secure:   true,
	})
	http.Redirect(w, r, "/", http.StatusFound)
	return nil
}
