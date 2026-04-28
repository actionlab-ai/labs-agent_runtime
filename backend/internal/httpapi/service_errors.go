package httpapi

import "errors"

type apiError struct {
	status int
	err    error
}

func (e *apiError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *apiError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func withStatus(status int, err error) error {
	if err == nil {
		return nil
	}
	return &apiError{status: status, err: err}
}

func errorStatus(err error, fallback int) int {
	var apiErr *apiError
	if errors.As(err, &apiErr) && apiErr.status > 0 {
		return apiErr.status
	}
	return fallback
}
