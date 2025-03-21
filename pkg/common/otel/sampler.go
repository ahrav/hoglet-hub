package otel

import (
	"strings"

	"go.opentelemetry.io/otel/sdk/trace"
)

type endpointExcluder struct {
	endpoints   map[string]struct{}
	probability float64
}

func newEndpointExcluder(endpoints map[string]struct{}, probability float64) endpointExcluder {
	return endpointExcluder{
		endpoints:   endpoints,
		probability: probability,
	}
}

// ShouldSample implements the sampler interface. It prevents the specified
// endpoints from being added to the trace.
func (ee endpointExcluder) ShouldSample(parameters trace.SamplingParameters) trace.SamplingResult {
	for i := range parameters.Attributes {
		if parameters.Attributes[i].Key == "http.target" {
			path := parameters.Attributes[i].Value.AsString()

			// Check for exact matches
			if _, exists := ee.endpoints[path]; exists {
				return trace.SamplingResult{Decision: trace.Drop}
			}

			// Check for prefix matches (for paths like /debug/*)
			for endpoint := range ee.endpoints {
				if strings.HasPrefix(path, endpoint) {
					return trace.SamplingResult{Decision: trace.Drop}
				}
			}
		}
	}

	return trace.TraceIDRatioBased(ee.probability).ShouldSample(parameters)
}

// Description implements the sampler interface.
func (endpointExcluder) Description() string { return "customSampler" }
