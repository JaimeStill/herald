package middleware

import (
	"os"
	"strconv"
	"strings"
)

// CORSConfig holds CORS policy settings.
type CORSConfig struct {
	Enabled          bool     `toml:"enabled"`
	Origins          []string `toml:"origins"`
	AllowedMethods   []string `toml:"allowed_methods"`
	AllowedHeaders   []string `toml:"allowed_headers"`
	AllowCredentials bool     `toml:"allow_credentials"`
	MaxAge           int      `toml:"max_age"`
}

// CORSEnv maps CORS config fields to environment variable names for override injection.
type CORSEnv struct {
	Enabled          string
	Origins          string
	AllowedMethods   string
	AllowedHeaders   string
	AllowCredentials string
	MaxAge           string
}

// Finalize applies defaults and environment variable overrides.
func (c *CORSConfig) Finalize(env *CORSEnv) error {
	c.loadDefaults()
	if env != nil {
		c.loadEnv(env)
	}
	return nil
}

// Merge overwrites fields from overlay. Boolean fields always apply; slice and int
// fields only apply when non-zero.
func (c *CORSConfig) Merge(overlay *CORSConfig) {
	c.Enabled = overlay.Enabled
	c.AllowCredentials = overlay.AllowCredentials

	if overlay.Origins != nil {
		c.Origins = overlay.Origins
	}
	if overlay.AllowedMethods != nil {
		c.AllowedMethods = overlay.AllowedMethods
	}
	if overlay.AllowedHeaders != nil {
		c.AllowedHeaders = overlay.AllowedHeaders
	}
	if overlay.MaxAge >= 0 {
		c.MaxAge = overlay.MaxAge
	}
}

func (c *CORSConfig) loadDefaults() {
	if len(c.AllowedMethods) == 0 {
		c.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if len(c.AllowedHeaders) == 0 {
		c.AllowedHeaders = []string{"Content-Type", "Authorization"}
	}
	if c.MaxAge <= 0 {
		c.MaxAge = 3600
	}
}

func (c *CORSConfig) loadEnv(env *CORSEnv) {
	if env.Enabled != "" {
		if v := os.Getenv(env.Enabled); v != "" {
			if enabled, err := strconv.ParseBool(v); err == nil {
				c.Enabled = enabled
			}
		}
	}
	if env.Origins != "" {
		if v := os.Getenv(env.Origins); v != "" {
			origins := strings.Split(v, ",")
			c.Origins = make([]string, 0, len(origins))
			for _, origin := range origins {
				if trimmed := strings.TrimSpace(origin); trimmed != "" {
					c.Origins = append(c.Origins, trimmed)
				}
			}
		}
	}
	if env.AllowedMethods != "" {
		if v := os.Getenv(env.AllowedMethods); v != "" {
			methods := strings.Split(v, ",")
			c.AllowedMethods = make([]string, 0, len(methods))
			for _, method := range methods {
				if trimmed := strings.TrimSpace(method); trimmed != "" {
					c.AllowedMethods = append(c.AllowedMethods, trimmed)
				}
			}
		}
	}
	if env.AllowedHeaders != "" {
		if v := os.Getenv(env.AllowedHeaders); v != "" {
			headers := strings.Split(v, ",")
			c.AllowedHeaders = make([]string, 0, len(headers))
			for _, header := range headers {
				if trimmed := strings.TrimSpace(header); trimmed != "" {
					c.AllowedHeaders = append(c.AllowedHeaders, trimmed)
				}
			}
		}
	}
	if env.AllowCredentials != "" {
		if v := os.Getenv(env.AllowCredentials); v != "" {
			if creds, err := strconv.ParseBool(v); err == nil {
				c.AllowCredentials = creds
			}
		}
	}
	if env.MaxAge != "" {
		if v := os.Getenv(env.MaxAge); v != "" {
			if maxAge, err := strconv.Atoi(v); err == nil {
				c.MaxAge = maxAge
			}
		}
	}
}
