package mcp

import (
	"errors"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
)

// mapError translates a use-case error into a JSON-RPC error; unrecognized errors are internal (-32603).
func mapError(err error) *RPCError {
	switch {
	case errors.Is(err, apperrs.ErrNotFound):
		return &RPCError{Code: CodeNotFound, Message: err.Error()}
	case errors.Is(err, apperrs.ErrUnauthorized):
		return &RPCError{Code: CodeUnauthorized, Message: err.Error()}
	case errors.Is(err, apperrs.ErrForbidden):
		return &RPCError{Code: CodeUnauthorized, Message: err.Error()}
	case errors.Is(err, apperrs.ErrInvalid):
		return &RPCError{Code: CodeInvalidParams, Message: err.Error()}
	default:
		return &RPCError{Code: CodeInternalError, Message: err.Error()}
	}
}
