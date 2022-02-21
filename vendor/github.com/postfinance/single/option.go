package single

// Option configures Single
type Option func(*Single)

// WithLockPath configures the path for the lockfile
func WithLockPath(lockpath string) Option {
	return func(s *Single) {
		s.path = lockpath
	}
}
