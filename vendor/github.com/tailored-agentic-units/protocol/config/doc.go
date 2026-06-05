// Package config provides configuration management for TAU agent infrastructure.
// It defines structures for agent, model, provider, and client configuration
// with support for human-readable duration strings and JSON serialization.
//
// Configuration supports layered composition through Merge methods,
// enabling defaults to be overridden by file-loaded or runtime configuration.
package config
