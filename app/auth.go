package app

import (
	"errors"
	"net/http"

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
