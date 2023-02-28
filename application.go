package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mateusz834/charts/service"
	"github.com/mateusz834/charts/templates"
)

// errHandler is an http.HandlerFunc, but with an aditional error return.
// nil return means that the response was sent, any error means that
// some error happend and that respose was not been sent.
type errHandler func(w http.ResponseWriter, r *http.Request) error

type httpError struct {
	JSONBody     any
	ResponseCode int
	DebugErr     error
}

func (e *httpError) Error() string {
	return fmt.Sprintf("http error: %v, caused by: %v", http.StatusText(e.ResponseCode), e.DebugErr)
}

// afterWriteHeaderError is an error ityep that should be returned after handler
// called the WriteHeader method, so that the errhandler does not call WriteHeader again.
type afterWriteHeaderError struct {
	Err error

	// ConnectionError true means that this is a connection related
	// error (error returned from ResponseWriter's Write method)
	ConnectionError bool
}

func (e *afterWriteHeaderError) Error() string { return e.Err.Error() }

// Handler returns an handler that converts errHandler to a normal http handler,
// any error that happens in h is logged and an InternalServerError is send (without body).
// Response code an the body can be controlled with *httpError.
func (h errHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			switch v := err.(type) {
			case *httpError:
				if v.JSONBody != nil {
					w.Header().Set("Content-Type", "application/json")
				}
				w.WriteHeader(v.ResponseCode)
				if v.JSONBody != nil {
					if err := json.NewEncoder(w).Encode(v.JSONBody); err != nil {
						log.Printf("error while encoding error json body: %v", err)
					}
				}
				log.Printf("debug error: %v", v.DebugErr)
			case *afterWriteHeaderError:
				// TODO: better logging, and use ConnectionError to pioritize the logging mechanism,
				// if connectonerror == true, then it it should be like debug, if not then error.
				if v.ConnectionError {
					log.Printf("connection error: %v", err)
				} else {
					log.Printf("error: %v", err)
				}
			default:
				w.WriteHeader(http.StatusInternalServerError)
				log.Printf("error: %v", err)
			}
		}
	}
}

type writerErrorWrapper struct{ w io.Writer }
type writeError struct{ error }

func (w writerErrorWrapper) Write(buf []byte) (int, error) {
	num, err := w.w.Write(buf)
	if err != nil {
		return num, writeError{err}
	}
	return num, nil
}

func sendJSON(w http.ResponseWriter, status int, content any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(writerErrorWrapper{w}).Encode(content)
	if err != nil {
		if v, ok := err.(writeError); ok {
			return &afterWriteHeaderError{Err: v.error, ConnectionError: true}
		}
		return &afterWriteHeaderError{Err: err}
	}
	return nil
}

type SessionService interface {
	NewSession(githubUserID uint64) (string, error)
	IsSessionValid(session string) (uint64, bool, error)
}

type PublicSharesService interface {
	IsPathAvail(path string) (bool, error)
	CreateShare(req *service.CreateShare) (string, error)
	GetShare(path string) (*service.Share, error)
}

type application struct {
	githubOAuth         oauth
	sessionService      SessionService
	publicSharesService PublicSharesService
}

func NewApplication(oauth oauth, session SessionService, publicShares PublicSharesService) *application {
	return &application{
		githubOAuth:         oauth,
		sessionService:      session,
		publicSharesService: publicShares,
	}
}

func (a *application) start() error {
	mux := http.NewServeMux()
	a.setRoutes(mux)
	return http.ListenAndServe("localhost:8888", mux)
}

func (a *application) setRoutes(mux *http.ServeMux) {
	// TODO: require specific HTTP methods (in middlewares).

	mux.Handle("/assets/", http.FileServer(http.FS(assets)))
	mux.Handle("/s/", errHandler(a.shareVisit).Handler())

	mux.Handle("/github-login-callback", errHandler(a.githubLoginCallback).Handler())
	mux.Handle("/validate-path", errHandler(a.validatePath).Handler())

	mux.Handle("/create-share", a.authMiddleware(a.createShare).Handler())
	mux.Handle("/is-authenticated", a.authMiddleware(func(w http.ResponseWriter, _ *http.Request) error {
		return sendJSON(w, http.StatusOK, struct{}{})
	}).Handler())

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		templates.Index(w)
	})
}

type githubUserIDKey uint8

func (a *application) authMiddleware(handler errHandler) errHandler {
	type errResponse struct {
		Error string `json:"error"`
	}

	return func(w http.ResponseWriter, r *http.Request) error {
		cookie, err := r.Cookie("__Host-session")
		if err != nil {
			return sendJSON(w, http.StatusOK, errResponse{Error: "missing valid session cookie"})
		}

		githubUserID, v, err := a.sessionService.IsSessionValid(cookie.Value)
		if err != nil {
			// TODO: here we cause internalServerError, we shouldn't return it always.
			// e.g. invalid session cookie should not cause it.
			return err
		}

		if !v {
			return sendJSON(w, http.StatusOK, errResponse{Error: "invalid session cookie"})
		}

		return handler(w, r.WithContext(context.WithValue(context.Background(), githubUserIDKey(0), githubUserID)))
	}
}

func githubUserID(r *http.Request) uint64 {
	return r.Context().Value(githubUserIDKey(0)).(uint64)
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
			ResponseCode: http.StatusBadGateway,
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

func (a *application) validatePath(w http.ResponseWriter, r *http.Request) error {
	reqBody := struct {
		Path string `json:"path"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		return &httpError{ResponseCode: http.StatusBadGateway, DebugErr: err}
	}

	type response struct {
		Avail  bool   `json:"avail"`
		Reason string `json:"reason,omitempty"`
	}

	avail, err := a.publicSharesService.IsPathAvail(reqBody.Path)
	if err != nil {
		var publicErr service.PublicError
		if errors.As(err, &publicErr) {
			return sendJSON(w, http.StatusOK, response{Avail: false, Reason: err.Error()})
		}
		return err
	}

	return sendJSON(w, http.StatusOK, response{Avail: avail})
}

func (a *application) createShare(w http.ResponseWriter, r *http.Request) error {
	reqBody := struct {
		CustomPath *string `json:"custom_path"`
		Chart      string  `json:"chart"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		return &httpError{ResponseCode: http.StatusBadGateway, DebugErr: err}
	}

	createShare := &service.CreateShare{
		EncodedChart: reqBody.Chart,
		GithubUserID: githubUserID(r),
	}

	if reqBody.CustomPath != nil {
		createShare.Path = *reqBody.CustomPath
		createShare.CustomPath = true
	}

	path, err := a.publicSharesService.CreateShare(createShare)
	if err != nil {
		var publicErr service.PublicError
		if errors.As(err, &publicErr) || err == service.ErrPathUnavail {
			type errResponse struct {
				Error string `json:"error"`
			}
			return sendJSON(w, http.StatusOK, errResponse{Error: err.Error()})
		}
		return err
	}

	type response struct {
		Path string `json:"path"`
	}

	return sendJSON(w, http.StatusOK, response{Path: path})
}

func (a *application) shareVisit(w http.ResponseWriter, r *http.Request) error {
	sharePath := strings.TrimPrefix(r.URL.Path, "/s/")
	share, err := a.publicSharesService.GetShare(sharePath)
	if err != nil {
		return err
	}
	http.Redirect(w, r, "/?s="+share.EncodedChart, http.StatusFound)
	return nil
}
