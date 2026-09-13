package cli

// usageError marks a failure caused by incorrect use of the CLI itself
// (wrong number of arguments, an invalid flag value) rather than a business
// failure of the use case. It maps to the same exit code Cobra would use by
// default for such errors (contracts/cli.md) — which this package's
// SilenceErrors/SilenceUsage settings would otherwise suppress, since every
// error is now presented and translated to an exit code by the composition
// root instead.
type usageError struct {
	err error
}

func newUsageError(err error) error {
	return &usageError{err: err}
}

func (e *usageError) Error() string {
	return e.err.Error()
}

func (e *usageError) Unwrap() error {
	return e.err
}
