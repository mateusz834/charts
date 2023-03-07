package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/mateusz834/charts/chart"
	"github.com/mateusz834/charts/storage"
)

type SharesStorage interface {
	IsPathAvail(path string) (bool, error)
	CreateShare(share *storage.Share) (bool, error)
	GetShare(path string) (*storage.Share, error)
	GetUserShares(githubUserID uint64) ([]storage.Share, error)
	RemoveShare(path string, githubUserID uint64) error
}

type SharesService struct {
	storage SharesStorage
}

func NewSharesService(storage SharesStorage) SharesService {
	return SharesService{
		storage: storage,
	}
}

var errInvalidPath = errors.New("use a-z,A-Z,0-9,'-' characters only")
var errPathTooShort = errors.New("url must be at at least 4 characters long")
var errPathTooLong = errors.New("url must be at most 48 characters long")

func (s *SharesService) isPathValid(path string) error {
	if len(path) < 4 {
		return PublicWrapperError{errPathTooShort}
	}

	if len(path) > 48 {
		return PublicWrapperError{errPathTooLong}
	}

	for _, v := range path {
		if !(v >= 'a' && v <= 'z' || v >= 'A' && v <= 'Z' || v >= '0' && v <= '9' || v == '-') {
			return PublicWrapperError{errInvalidPath}
		}
	}

	return nil
}

func (s *SharesService) IsPathAvail(path string) (bool, error) {
	if err := s.isPathValid(path); err != nil {
		return false, err
	}

	return s.storage.IsPathAvail(path)
}

type CreateShare struct {
	GithubUserID uint64
	CustomPath   bool
	Path         string
	EncodedChart string
}

var ErrPathUnavail = errors.New("path is not available")

type CreateShareError struct {
	Type string
	Err  error
}

func (c *CreateShareError) Error() string { return c.Err.Error() }

func (s *SharesService) CreateShare(req *CreateShare) (string, error) {
	var path string
	if req.CustomPath {
		if err := s.isPathValid(req.Path); err != nil {
			return "", &CreateShareError{"path", err}
		}
		path = req.Path
	} else {
		pathBin := make([]byte, 8)
		if _, err := rand.Read(pathBin); err != nil {
			return "", err
		}
		path = base64.RawURLEncoding.EncodeToString(pathBin)
	}

	chart, err := chart.Decode(req.EncodedChart)
	if err != nil {
		return "", &CreateShareError{"chart", err}
	}

	avail, err := s.storage.CreateShare(&storage.Share{
		GithubUserID: req.GithubUserID,
		Path:         path,
		Chart:        chart,
	})

	if err != nil {
		return "", err
	}

	if !avail {
		return "", &CreateShareError{"path", ErrPathUnavail}
	}

	return path, nil
}

type Share struct {
	GithubUserID uint64
	Path         string
	EncodedChart string
}

func (s *SharesService) GetShare(path string) (*Share, error) {
	share, err := s.storage.GetShare(path)
	if err != nil {
		return nil, err
	}
	encoded, err := chart.Encode(share.Chart)
	if err != nil {
		return nil, err
	}
	return &Share{
		GithubUserID: share.GithubUserID,
		Path:         path,
		EncodedChart: encoded,
	}, nil
}

func (s *SharesService) GetAllUserShares(githubUserID uint64) ([]Share, error) {
	shares, err := s.storage.GetUserShares(githubUserID)
	if err != nil {
		return nil, err
	}

	res := make([]Share, len(shares))
	for i, v := range shares {
		encodedChart, err := chart.Encode(v.Chart)
		if err != nil {
			return nil, err
		}

		res[i] = Share{
			GithubUserID: v.GithubUserID,
			Path:         v.Path,
			EncodedChart: encodedChart,
		}
	}

	return res, nil
}

func (s *SharesService) RemoveShare(path string, githubUserID uint64) error {
	return s.storage.RemoveShare(path, githubUserID)
}
