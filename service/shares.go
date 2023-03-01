package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/mateusz834/charts/chart"
	"github.com/mateusz834/charts/storage"
)

var ErrNotFound = storage.ErrNotFound

type SharesStorage interface {
	IsPathAvail(path string) (bool, error)
	CreateShare(share *storage.Share) (bool, error)
	GetShare(path string) (*storage.Share, error)
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
		return PublicError{errPathTooShort}
	}

	if len(path) > 48 {
		return PublicError{errPathTooLong}
	}

	for _, v := range path {
		if !(v >= 'a' && v <= 'z' || v >= 'A' && v <= 'Z' || v >= '0' && v <= '9' || v == '-') {
			return PublicError{errInvalidPath}
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

func (s *SharesService) CreateShare(req *CreateShare) (string, error) {
	var path string
	if req.CustomPath {
		if err := s.isPathValid(req.Path); err != nil {
			return "", err
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
		return "", err
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
		return "", ErrPathUnavail
	}

	return path, nil
}

type Share struct {
	GithubUserID uint64
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
		EncodedChart: encoded,
	}, nil
}
