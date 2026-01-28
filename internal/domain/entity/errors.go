package entity

import "errors"

var (
	ErrMissingUser   = errors.New("missing required field: user")
	ErrMissingAsset  = errors.New("missing required field: asset")
	ErrMissingAmount = errors.New("missing required field: amount")
)
