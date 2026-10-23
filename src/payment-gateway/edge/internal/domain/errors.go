package domain

import (
	"errors"
	"fmt"
)

var (
	ErrInvalid      = errors.New("invalid request")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrGone         = errors.New("gone")
	ErrUnavailable  = errors.New("unavailable")
	ErrTimeout      = errors.New("timeout")
	ErrInternal     = errors.New("internal error")
	ErrInvalidCard  = fmt.Errorf("%w: invalid card", ErrInvalid)
)

type Error struct {
	Kind   error
	Detail string
}

func (e *Error) Error() string { return e.Kind.Error() + ": " + e.Detail }

func (e *Error) Unwrap() error { return e.Kind }

func New(kind error, detail string) error { return &Error{Kind: kind, Detail: detail} }

func Invalid(detail string) error { return New(ErrInvalid, detail) }

func InvalidDetail(err error) string {
	var domainErr *Error
	if errors.As(err, &domainErr) && errors.Is(domainErr.Kind, ErrInvalid) {
		return domainErr.Detail
	}
	return "invalid request"
}
