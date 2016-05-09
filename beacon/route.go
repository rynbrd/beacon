package beacon

// NewRoute creates a route from the provided filter and backend.
func NewRoute(filter Filter, backend Backend) Route {
	return struct {
		Filter
		Backend
	}{
		filter,
		backend,
	}
}

// Route processes events which match a particular filter pattern.
type Route interface {
	Filter
	Backend
}
