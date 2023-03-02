package service

import "github.com/mateusz834/charts/storage"

var ErrNotFound = storage.ErrNotFound

type PublicError interface {
	PublicError() string
}

// PublicWrapperError it is an error that may be triggered by users,
// is safe to expose this error publicly.
type PublicWrapperError struct {
	Err error
}

func (e PublicWrapperError) Error() string {
	return e.Err.Error()
}

func (e PublicWrapperError) PublicError() string {
	return e.Err.Error()
}

// PublicWithDebugError is is similar to PublicWrapperError, but also contains a debug
// error. You can think of this like: Public error was caused by a Debug error, but
// we don't want to expose the debug error publicly.
type PublicWithDebugError struct {
	Public string
	Debug  error
}

func (e PublicWithDebugError) Error() string {
	return e.Public + ": " + e.Debug.Error()
}

func (e PublicWithDebugError) PublicError() string {
	return e.Public
}
