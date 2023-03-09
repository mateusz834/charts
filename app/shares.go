package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/mateusz834/charts/service"
)

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

	createShare := &service.CreateShare{
		EncodedChart: reqBody.Chart,
		GithubUserID: a.getGithubUserID(r),
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

func (a *application) shareInfo(w http.ResponseWriter, r *http.Request) error {
	sharePath := strings.TrimPrefix(r.URL.Path, "/share/")
	share, err := a.publicSharesService.GetShare(sharePath)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			return sendJSON(w, http.StatusOK, struct{}{})
		}
		return err
	}

	type response struct {
		Chart        string `json:"chart"`
		GithubUserID uint64 `json:"github_user_id"`
	}

	return sendJSON(w, http.StatusOK, response{
		Chart:        share.EncodedChart,
		GithubUserID: share.GithubUserID,
	})
}

func (a *application) getAllUserShares(w http.ResponseWriter, r *http.Request) error {
	shares, err := a.publicSharesService.GetAllUserShares(a.getGithubUserID(r))
	if err != nil {
		return err
	}

	type share struct {
		Path  string `json:"path"`
		Chart string `json:"chart"`
	}

	res := make([]share, len(shares))
	for i, v := range shares {
		res[i] = share{Path: v.Path, Chart: v.EncodedChart}
	}

	return sendJSON(w, http.StatusOK, res)
}

func (a *application) removeChart(w http.ResponseWriter, r *http.Request) error {
	reqBody := struct {
		Path string `json:"path"`
	}{}

	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		return &httpError{ResponseCode: http.StatusBadRequest, DebugErr: err}
	}

	if err := a.publicSharesService.RemoveShare(reqBody.Path, a.getGithubUserID(r)); err != nil {
		return err
	}

	return sendJSON(w, http.StatusOK, struct{}{})
}
