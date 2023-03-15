package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/mateusz834/charts/service"
)

type githubUserIDKey uint8

func (a *application) auth(handler errHandler) errHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		githubUserID, err := a.authenticate(r)
		if err != nil {
			var publicError service.PublicError
			if errors.As(err, &publicError) {
				type errResponse struct {
					ErrorType string `json:"error_type"`
					ErrorMsg  string `json:"error_msg"`
				}
				if err := sendJSON(w, http.StatusOK, errResponse{
					ErrorType: "auth",
					ErrorMsg:  publicError.PublicError(),
				}); err != nil {
					return err
				}
				return &debugError{err}
			}
			return err
		}
		return handler(w, r.WithContext(context.WithValue(r.Context(), githubUserIDKey(0), githubUserID)))
	}
}

func (a *application) getGithubUserID(r *http.Request) uint64 {
	return r.Context().Value(githubUserIDKey(0)).(uint64)
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
	query.Add("client_id", a.githubOAuth.ClientID)
	query.Add("state", csrf)
	query.Add("scope", "")
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

func (a *application) userInfo(w http.ResponseWriter, r *http.Request) error {
	type response struct {
		GithubUserID uint64 `json:"github_user_id"`
	}
	return sendJSON(w, http.StatusOK, response{GithubUserID: a.getGithubUserID(r)})
}

func (a *application) logout(w http.ResponseWriter, r *http.Request) error {
	if s, err := r.Cookie("__Host-session"); err == nil {
		if err := a.sessionService.RemoveSession(s.Value); err != nil {
			if errors.As(err, new(service.PublicError)) {
				w.WriteHeader(http.StatusBadRequest)
				return &debugError{err}
			}
			return err
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "__Host-session",
		Path:    "/",
		Expires: time.UnixMicro(0),
		MaxAge:  -1,
		Secure:  true,
	})
	http.Redirect(w, r, "/", http.StatusFound)
	return nil
}
