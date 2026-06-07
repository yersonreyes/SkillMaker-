// Package service contains the business logic for the notifications module.
package service

import "errors"

// ErrNotFound is returned when a notification does not exist or does not
// belong to the requesting user (caller-scoped guard).
var ErrNotFound = errors.New("notification not found")
