package portability

import "errors"

var (
	ErrInvalid      = errors.New("portability: invalid argument")
	ErrMalformed    = errors.New("portability: malformed stream")
	ErrTruncated    = errors.New("portability: truncated stream")
	ErrIO           = errors.New("portability: I/O failure")
	ErrIncompatible = errors.New("portability: incompatible archive")
	ErrLimit        = errors.New("portability: limit exceeded")
	ErrIntegrity    = errors.New("portability: integrity failure")
	ErrSequence     = errors.New("portability: sequence mismatch")
	ErrSink         = errors.New("portability: consumer sink failure")
	ErrIncomplete   = errors.New("portability: archive incomplete")
	ErrFinalized    = errors.New("portability: already finalized")
	ErrComplete     = errors.New("portability: archive complete")
)

type classifiedError struct {
	class error
	op    string
	cause error
}

// Error intentionally omits the underlying cause text because consumer I/O
// errors can contain payload fragments or storage identifiers.
// Complexity: time O(1), Omega(1), tight Theta(1); auxiliary space O(1),
// Omega(1), tight Theta(1); the returned string may reuse existing storage.
func (e *classifiedError) Error() string {
	if e.op == "" {
		return e.class.Error()
	}
	return e.class.Error() + ": " + e.op
}

// Unwrap returns only the stable library classification. Callback and I/O
// causes are deliberately excluded from errors.Is traversal.
// Complexity: time O(1), Omega(1), tight Theta(1); auxiliary space O(1),
// Omega(1), tight Theta(1).
func (e *classifiedError) Unwrap() error { return e.class }

// Cause returns the raw underlying callback or I/O error explicitly. It may
// contain sensitive consumer data and is not safe to log or display without
// caller-controlled redaction.
// Complexity: time and auxiliary space O(1), Omega(1), tight Theta(1).
func (e *classifiedError) Cause() error { return e.cause }

// wrap creates a classified error whose Error string is redaction-safe.
// Complexity: time O(1), Omega(1), tight Theta(1); auxiliary space O(1),
// Omega(1), tight Theta(1), with one error allocation.
func wrap(class error, op string, cause error) error {
	return &classifiedError{class: class, op: op, cause: cause}
}
