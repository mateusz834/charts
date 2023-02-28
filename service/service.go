package service

// PublicError is an error that is safe to expose publicly.
type PublicError struct {
	Err error
}

func (e PublicError) Error() string {
	return e.Err.Error()
}

func (e PublicError) Unwrap() error {
	return e.Err
}
