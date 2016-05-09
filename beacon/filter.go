package beacon

// NewFilter creates a filter from the provided pattern. The pattern has the
// form 'label1=value1,label2=value2,...'. The container must match all of the
// lable/value pairs. Only matching against labels is currently supported.
func NewFilter(pattern string) (Filter, error) {
}

// Filter is used to match containers againston a set of criteria.
type Filter interface {
	MatchContainer(*Container) bool
}
