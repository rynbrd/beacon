package beacon

import (
	"fmt"
	"strings"
)

// NewFilter creates a filter from the provided pattern. The pattern has the
// form 'label1=value1,label2=value2,...'. The container must match all of the
// lable/value pairs. Only matching against labels is currently supported.
func NewFilter(pattern string) (Filter, error) {
	if pattern == "" {
		return &labelFilter{}, nil
	}
	pairs := strings.Split(pattern, ",")
	labels := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) > 1 {
			labels[parts[0]] = parts[1]
		} else {
			return nil, fmt.Errorf("invalid filter pattern: %s", pattern)
		}
	}
	return &labelFilter{labels: labels}, nil
}

// Filter is used to match containers againston a set of criteria.
type Filter interface {
	MatchContainer(*Container) bool
}

// Basic filter which checks that the container has all of the given label values.
type labelFilter struct {
	labels map[string]string
}

func (f *labelFilter) MatchContainer(c *Container) bool {
	for label, value1 := range f.labels {
		if value2, ok := c.Labels[label]; !ok || value1 != value2 {
			return false
		}
	}
	return true
}
