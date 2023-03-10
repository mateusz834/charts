package app

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"

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
	RemoveSession(session string) error
}

type PublicSharesService interface {
	IsPathAvail(path string) (bool, error)
	CreateShare(req *service.CreateShare) (string, error)
	GetShare(path string) (*service.Share, error)
	GetAllUserShares(githubUserID uint64) ([]service.Share, error)
	RemoveShare(path string, githubUserID uint64) error
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

func httpMethod(method string, handler errHandler) errHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		if r.Method != method {
			w.Header().Add("Allow", method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return nil
		}
		return handler(w, r)
	}
}

func requireJSONContentType(handler errHandler) errHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		mimetype, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || mimetype != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return nil
		}
		return handler(w, r)
	}
}

func (a *application) setRoutes(mux *http.ServeMux) {
	mux.Handle("/assets/", http.FileServer(http.FS(assets)))

	mux.HandleFunc("/share/", httpMethod(http.MethodGet, a.shareInfo).Handler())
	mux.HandleFunc("/s/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		// TODO: same thing here as with sendJSON for error handling.
		templates.Share(w)
	})

	mux.Handle("/github-login", httpMethod(http.MethodGet, a.githubLogin).Handler())
	mux.Handle("/github-login-callback", httpMethod(http.MethodGet, a.githubLoginCallback).Handler())

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
	mux.Handle("/create-share",
		httpMethod(
			http.MethodPost,
			requireJSONContentType(a.auth(a.createShare)),
		).Handler(),
	)

	// Accepts a JSON: { "path": "custom_path" }, checks whether this
	// path is valid and whether it is avaliable for creation of a new share.
	// Returns (200 OK) with one of following:
	// 1) { "avail": true }
	// 2) { "avail": false, "cause": "cause message" }
	mux.Handle("/validate-path",
		httpMethod(
			http.MethodPost,
			requireJSONContentType(a.validatePath),
		).Handler(),
	)

	// Returns (200 OK) with JSON:
	// (on sucess) { "github_user_id": 1000 }
	// (on error) { "error_type": "error_type", error_msg: "error msg" }
	// error_type is one of following:
	// - "auth" -> authentication error (probaly expired), so user is not authenticated.
	mux.Handle("/user-info", httpMethod(http.MethodPost, a.auth(a.userInfo)).Handler())

	// Accepts JSON: { "path": "path" }
	// Return (200 OK) with one following responses:
	// (on success) {} (empty json)
	// (on error) { "error_type": "error_type", "error_msg": "error_msg" }
	// error_type is one of following:
	// - "auth" -> authenticated error
	mux.Handle("/remove-chart", httpMethod(http.MethodPost,
		requireJSONContentType(a.auth(a.removeChart)),
	).Handler())

	mux.Handle("/get-all-user-shares", httpMethod(http.MethodGet, a.auth(a.getAllUserShares)).Handler())
	mux.Handle("/logout", httpMethod(http.MethodGet, a.logout).Handler())

	mux.HandleFunc("/my-shares", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		// TODO: same thing here as with sendJSON for error handling.
		templates.MyShares(w)
	})

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
