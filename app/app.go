package app

import (
	"embed"
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

//go:embed assets
var assets embed.FS

// errHandler is an http.HandlerFunc, but with an aditional error return.
// nil return means that the response was sent, any error means that
// some error happend and that respose was not been sent.
type errHandler func(w http.ResponseWriter, r *http.Request) error

type httpError struct {
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

type debugError struct {
	Err error
}

func (e *debugError) Error() string { return e.Err.Error() }

// Handler returns an handler that converts errHandler to a normal http handler,
// any error that happens in h is logged and an InternalServerError is send (without body).
// Response code an the body can be controlled with *httpError.
func (h errHandler) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			switch v := err.(type) {
			case *httpError:
				w.WriteHeader(v.ResponseCode)
				log.Printf("debug error: %v", v.DebugErr)
			case *afterWriteHeaderError:
				// TODO: better logging, and use ConnectionError to pioritize the logging mechanism,
				// if connectonerror == true, then it it should be like debug, if not then error.
				if v.ConnectionError {
					log.Printf("connection error: %v", err)
				} else {
					log.Printf("error: %v", err)
				}
			case *debugError:
				log.Printf("debug error: %v", err)
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
	IsSessionValid(session string) (uint64, error)
}

type PublicSharesService interface {
	IsPathAvail(path string) (bool, error)
	CreateShare(req *service.CreateShare) (string, error)
	GetShare(path string) (*service.Share, error)
}

type application struct {
	githubOAuth         OAuth
	sessionService      SessionService
	publicSharesService PublicSharesService
}

func NewApplication(oauth OAuth, session SessionService, publicShares PublicSharesService) *application {
	return &application{
		githubOAuth:         oauth,
		sessionService:      session,
		publicSharesService: publicShares,
	}
}

func (a *application) Start() error {
	mux := http.NewServeMux()
	a.setRoutes(mux)
	return http.ListenAndServe("localhost:8888", mux)
}

func (a *application) setRoutes(mux *http.ServeMux) {
	// TODO: require specific HTTP methods (in middlewares).

	mux.Handle("/assets/", http.FileServer(http.FS(assets)))
	mux.Handle("/s/", errHandler(a.shareVisit).Handler())

	mux.Handle("/github-login-callback", errHandler(a.githubLoginCallback).Handler())

	// Accepts a JSON in one of following forms:
	// 1) { "chart": "base64-encoded-chart" }, it will create a share with a server-generated path.
	// 2) { "chart": "base64-encoded-chart", custom_path: "path" }, it will create a share with
	// specified custom_path (if available).
	// Returns (200 OK) with JSON:
	// (on success) { "path": "custom_path or server-generated one" }
	// (on error) { "error_type": "error_type", error_msg: "error msg" }
	// error_type is one of following:
	// - "path" -> something is wrong with the custom_path (not available, not allowewd chars)
	// - "auth" -> authentication error (probaly expired), should ask the user to login again.
	// - "chart" -> error related to the provided chart encoding.
	mux.Handle("/create-share", errHandler(a.createShare).Handler())

	// Accepts a JSON: { "path": "custom_path" }, checks whether this
	// path is valid and whether it is avaliable for creation of a new share.
	// Returns (200 OK) with one of following:
	// 1) { "avail": true }
	// 2) { "avail": false, "cause": "cause message" }
	mux.Handle("/validate-path", errHandler(a.validatePath).Handler())

	// Returns (200 OK) { "authenticated": false } or { "authenticated": true }, depending on whether the request
	// contains a valid session cookie.
	mux.Handle("/is-authenticated", errHandler(a.isAuthenticated).Handler())

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		// TODO: same thing here as with sendJSON for error handling.
		templates.Index(w)
	})
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
		return &httpError{ResponseCode: http.StatusBadRequest, DebugErr: err}
	}

	type response struct {
		Avail bool   `json:"avail"`
		Cause string `json:"cause,omitempty"`
	}

	avail, err := a.publicSharesService.IsPathAvail(reqBody.Path)
	if err != nil {
		var publicErr service.PublicError
		if errors.As(err, &publicErr) {
			return sendJSON(w, http.StatusOK, response{Avail: false, Cause: publicErr.PublicError()})
		}
		return err
	}

	res := response{
		Avail: avail,
	}

	if !res.Avail {
		res.Cause = "url not available"
	}

	return sendJSON(w, http.StatusOK, res)
}

func (a *application) createShare(w http.ResponseWriter, r *http.Request) error {
	reqBody := struct {
		CustomPath *string `json:"custom_path"`
		Chart      string  `json:"chart"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		return &httpError{ResponseCode: http.StatusBadRequest, DebugErr: err}
	}

	type errResponse struct {
		ErrorType string `json:"error_type"`
		ErrorMsg  string `json:"error_msg"`
	}

	githubUserID, err := a.authenticate(r)
	if err != nil {
		var publicError service.PublicError
		if errors.As(err, &publicError) {
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

	createShare := &service.CreateShare{
		EncodedChart: reqBody.Chart,
		GithubUserID: githubUserID,
	}

	if reqBody.CustomPath != nil {
		createShare.Path = *reqBody.CustomPath
		createShare.CustomPath = true
	}

	path, err := a.publicSharesService.CreateShare(createShare)
	if err != nil {
		var createShareError *service.CreateShareError
		if errors.As(err, &createShareError) {
			return sendJSON(w, http.StatusOK, errResponse{
				ErrorType: createShareError.Type,
				ErrorMsg:  createShareError.Error(),
			})
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
		if errors.Is(err, service.ErrNotFound) {
			http.Redirect(w, r, "/", http.StatusFound)
			return nil
		}
		return err
	}
	http.Redirect(w, r, "/?s="+share.EncodedChart, http.StatusFound)
	return nil
}
