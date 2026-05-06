package config

// EnvironmentSource is implemented by any config struct that can hydrate
// itself from environment variables. Implementations conventionally delegate
// to nested EnvironmentSource children using env.AddPrefix to build the
// child's prefix from the parent's.
type EnvironmentSource interface {
	// FromEnv reads environment variables prefixed with prefix into the
	// receiver, leaving fields without a corresponding variable at their
	// existing values.
	FromEnv(prefix string)
}
