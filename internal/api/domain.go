package api

// Domain holds all domain systems that comprise the API.
type Domain struct{}

// NewDomain creates all domain systems from the API runtime.
func NewDomain(runtime *Runtime) *Domain {
	return &Domain{}
}
