package service

import "errors"

// ErrCertificateNotFound is returned when a certificate lookup finds no matching row,
// or when the requesting user does not own the certificate (anti-enumeration — like courses read path).
var ErrCertificateNotFound = errors.New("certificate not found")

// ErrNotOwner is returned by the service when a caller tries to access a certificate
// they do not own. Handlers map this to 404 (not 403) to avoid leaking existence.
var ErrNotOwner = errors.New("not the certificate owner")
