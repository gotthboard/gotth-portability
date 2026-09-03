package portability

import "errors"

var (
	ErrInvalid      = errors.New("portability: invalid argument")
	ErrMalformed    = errors.New("portability: malformed stream")
	ErrTruncated    = errors.New("portability: truncated stream")
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

// Unwrap returns both stable classification and programmatic cause without
// incorporating cause text in Error.
// Complexity: time O(1), Omega(1), tight Theta(1); auxiliary space O(1),
// Omega(1), tight Theta(1).
func (e *classifiedError) Unwrap() []error {
	if e.cause == nil {
		return []error{e.class}
	}
	return []error{e.class, e.cause}
}

// wrap creates a redaction-safe classified error.
// Complexity: time O(1), Omega(1), tight Theta(1); auxiliary space O(1),
// Omega(1), tight Theta(1), with one error allocation.
func wrap(class error, op string, cause error) error {
	return &classifiedError{class: class, op: op, cause: cause}
}
