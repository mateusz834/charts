package service

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/mateusz834/charts/storage"
)

type SessionStorage interface {
	StoreSession(s *storage.Session) error
	IsSessionValid(s *storage.Session) (bool, error)
}

type SessionService struct {
	storage SessionStorage
}

func NewSessionService(storage SessionStorage) SessionService {
	return SessionService{
		storage: storage,
	}
}

func (s *SessionService) NewSession(githubUserID uint64) (string, error) {
	ses := storage.Session{
		GithubUserID: githubUserID,
	}

	if _, err := rand.Read(ses.SessionID[:]); err != nil {
		return "", fmt.Errorf("failed to generate random session id: %v", err)
	}

	if err := s.storage.StoreSession(&ses); err != nil {
		return "", fmt.Errorf("failed to store session: %v", err)
	}

	return encodeSession(&ses), nil
}

func (s *SessionService) IsSessionValid(session string) (uint64, bool, error) {
	ses, err := decodeSession(session)
	if err != nil {
		return 0, false, err
	}
	avail, err := s.storage.IsSessionValid(ses)
	if err != nil {
		return 0, false, err
	}
	return ses.GithubUserID, avail, nil
}

func encodeSession(s *storage.Session) string {
	bin := make([]byte, 8+32)
	binary.BigEndian.PutUint64(bin[:8], s.GithubUserID)
	copy(bin[8:], s.SessionID[:])
	return base64.RawURLEncoding.EncodeToString(bin)
}

func decodeSession(s string) (*storage.Session, error) {
	bin := make([]byte, 8+32)
	if base64.RawURLEncoding.DecodedLen(len(s)) > len(bin) {
		return nil, errors.New("too long session value")
	}

	n, err := base64.RawURLEncoding.Decode(bin, []byte(s))
	if err != nil {
		return nil, fmt.Errorf("failed while decoding base64-encoded session: %v", err)
	}

	if n != len(bin) {
		return nil, errors.New("base64 decoded session has invalid length")
	}

	return &storage.Session{
		GithubUserID: binary.BigEndian.Uint64(bin[:8]),
		SessionID:    *(*[32]byte)(bin[8:]),
	}, nil
}
