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

type combinedError struct {
	outcome error
	cause   error
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

// Error returns only the joined redacted outcome text. Raw causes remain
// available solely through Cause.
// Complexity: delegated time and auxiliary space equal the joined outcomes.
func (e *combinedError) Error() string { return e.outcome.Error() }

// Unwrap exposes every redacted outcome for standard errors.Is traversal.
// Complexity: time and auxiliary space O(1), Omega(1), tight Theta(1).
func (e *combinedError) Unwrap() error { return e.outcome }

// Cause returns all non-nil raw causes as one standard multi-error.
// Complexity: time and auxiliary space O(1), Omega(1), tight Theta(1).
func (e *combinedError) Cause() error { return e.cause }

// wrap creates a classified error whose Error string is redaction-safe.
// Complexity: time O(1), Omega(1), tight Theta(1); auxiliary space O(1),
// Omega(1), tight Theta(1), with one error allocation.
func wrap(class error, op string, cause error) error {
	return &classifiedError{class: class, op: op, cause: cause}
}

// combine preserves two redacted outcomes and makes one Causer expose all of
// their non-nil raw causes. A nested combined cause remains traversable through
// the standard multi-error returned by errors.Join.
// Complexity: time and auxiliary space O(1), Omega(1), tight Theta(1), plus
// storage allocated by two errors.Join calls and one combined error.
func combine(primary, additional error) error {
	if primary == nil {
		return additional
	}
	if additional == nil {
		return primary
	}
	return &combinedError{
		outcome: errors.Join(primary, additional),
		cause:   errors.Join(rawCause(primary), rawCause(additional)),
	}
}

// rawCause extracts only an outcome's explicit cause, never its unwrap tree.
// Complexity: time and auxiliary space O(1), Omega(1), tight Theta(1).
func rawCause(err error) error {
	if causer, ok := err.(Causer); ok {
		return causer.Cause()
	}
	return nil
}
