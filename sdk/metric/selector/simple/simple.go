// Copyright 2019, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package simple // import "go.opentelemetry.io/otel/sdk/metric/selector/simple"

import (
	"go.opentelemetry.io/otel/api/core"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/array"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/counter"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/ddsketch"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/minmaxsumcount"
)

type (
	selectorInexpensive struct{}
	selectorExact       struct{}
	selectorSketch      struct {
		config *ddsketch.Config
	}
	selectorHistogram struct {
		boundaries []core.Number
	}
)

var (
	_ export.AggregationSelector = selectorInexpensive{}
	_ export.AggregationSelector = selectorSketch{}
	_ export.AggregationSelector = selectorExact{}
	_ export.AggregationSelector = selectorHistogram{}
)

// NewWithInexpensiveMeasure returns a simple aggregation selector
// that uses counter, minmaxsumcount and minmaxsumcount aggregators
// for the three kinds of metric.  This selector is faster and uses
// less memory than the others because minmaxsumcount does not
// aggregate quantile information.
func NewWithInexpensiveMeasure() export.AggregationSelector {
	return selectorInexpensive{}
}

// NewWithSketchMeasure returns a simple aggregation selector that
// uses counter, ddsketch, and ddsketch aggregators for the three
// kinds of metric.  This selector uses more cpu and memory than the
// NewWithInexpensiveMeasure because it uses one DDSketch per distinct
// measure/observer and labelset.
func NewWithSketchMeasure(config *ddsketch.Config) export.AggregationSelector {
	return selectorSketch{
		config: config,
	}
}

// NewWithExactMeasure returns a simple aggregation selector that uses
// counter, array, and array aggregators for the three kinds of metric.
// This selector uses more memory than the NewWithSketchMeasure
// because it aggregates an array of all values, therefore is able to
// compute exact quantiles.
func NewWithExactMeasure() export.AggregationSelector {
	return selectorExact{}
}

func NewWithHistogram(boundaries []core.Number) export.AggregationSelector {
	return selectorHistogram{boundaries}
}

func (selectorInexpensive) AggregatorFor(descriptor *export.Descriptor) export.Aggregator {
	switch descriptor.MetricKind() {
	case export.ObserverKind:
		fallthrough
	case export.MeasureKind:
		return minmaxsumcount.New(descriptor)
	default:
		return counter.New()
	}
}

func (s selectorSketch) AggregatorFor(descriptor *export.Descriptor) export.Aggregator {
	switch descriptor.MetricKind() {
	case export.ObserverKind:
		fallthrough
	case export.MeasureKind:
		return ddsketch.New(s.config, descriptor)
	default:
		return counter.New()
	}
}

func (selectorExact) AggregatorFor(descriptor *export.Descriptor) export.Aggregator {
	switch descriptor.MetricKind() {
	case export.ObserverKind:
		fallthrough
	case export.MeasureKind:
		return array.New()
	default:
		return counter.New()
	}
}

func (s selectorHistogram) AggregatorFor(descriptor *export.Descriptor) export.Aggregator {
	switch descriptor.MetricKind() {
	case export.ObserverKind:
		fallthrough
	case export.MeasureKind:
		return histogram.New(descriptor, s.boundaries)
	default:
		return counter.New()
	}
}
