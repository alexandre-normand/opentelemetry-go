package stdout_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/otel/api/core"
	"go.opentelemetry.io/otel/api/key"
	"go.opentelemetry.io/otel/exporters/metric/stdout"
	"go.opentelemetry.io/otel/exporters/metric/test"
	export "go.opentelemetry.io/otel/sdk/export/metric"
	"go.opentelemetry.io/otel/sdk/export/metric/aggregator"
	sdk "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/array"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/counter"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/ddsketch"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/lastvalue"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/minmaxsumcount"
	aggtest "go.opentelemetry.io/otel/sdk/metric/aggregator/test"
)

type testFixture struct {
	t        *testing.T
	ctx      context.Context
	exporter *stdout.Exporter
	output   *bytes.Buffer
}

func newFixture(t *testing.T, config stdout.Config) testFixture {
	buf := &bytes.Buffer{}
	config.Writer = buf
	config.DoNotPrintTime = true
	exp, err := stdout.NewRawExporter(config)
	if err != nil {
		t.Fatal("Error building fixture: ", err)
	}
	return testFixture{
		t:        t,
		ctx:      context.Background(),
		exporter: exp,
		output:   buf,
	}
}

func (fix testFixture) Output() string {
	return strings.TrimSpace(fix.output.String())
}

func (fix testFixture) Export(checkpointSet export.CheckpointSet) {
	err := fix.exporter.Export(fix.ctx, checkpointSet)
	if err != nil {
		fix.t.Error("export failed: ", err)
	}
}

func TestStdoutInvalidQuantile(t *testing.T) {
	_, err := stdout.NewRawExporter(stdout.Config{
		Quantiles: []float64{1.1, 0.9},
	})
	require.Error(t, err, "Invalid quantile error expected")
	require.Equal(t, aggregator.ErrInvalidQuantile, err)
}

func TestStdoutTimestamp(t *testing.T) {
	var buf bytes.Buffer
	exporter, err := stdout.NewRawExporter(stdout.Config{
		Writer:         &buf,
		DoNotPrintTime: false,
	})
	if err != nil {
		t.Fatal("Invalid config: ", err)
	}

	before := time.Now()

	checkpointSet := test.NewCheckpointSet(sdk.NewDefaultLabelEncoder())

	ctx := context.Background()
	desc := export.NewDescriptor("test.name", export.ObserverKind, nil, "", "", core.Int64NumberKind, false)
	lvagg := lastvalue.New()
	aggtest.CheckedUpdate(t, lvagg, core.NewInt64Number(321), desc)
	lvagg.Checkpoint(ctx, desc)

	checkpointSet.Add(desc, lvagg)

	if err := exporter.Export(ctx, checkpointSet); err != nil {
		t.Fatal("Unexpected export error: ", err)
	}

	after := time.Now()

	var printed map[string]interface{}

	if err := json.Unmarshal(buf.Bytes(), &printed); err != nil {
		t.Fatal("JSON parse error: ", err)
	}

	updateTS := printed["time"].(string)
	updateTimestamp, err := time.Parse(time.RFC3339Nano, updateTS)
	if err != nil {
		t.Fatal("JSON parse error: ", updateTS, ": ", err)
	}

	lastValueTS := printed["updates"].([]interface{})[0].(map[string]interface{})["time"].(string)
	lastValueTimestamp, err := time.Parse(time.RFC3339Nano, lastValueTS)
	if err != nil {
		t.Fatal("JSON parse error: ", lastValueTS, ": ", err)
	}

	require.True(t, updateTimestamp.After(before))
	require.True(t, updateTimestamp.Before(after))

	require.True(t, lastValueTimestamp.After(before))
	require.True(t, lastValueTimestamp.Before(after))

	require.True(t, lastValueTimestamp.Before(updateTimestamp))
}

func TestStdoutCounterFormat(t *testing.T) {
	fix := newFixture(t, stdout.Config{})

	checkpointSet := test.NewCheckpointSet(sdk.NewDefaultLabelEncoder())

	desc := export.NewDescriptor("test.name", export.CounterKind, nil, "", "", core.Int64NumberKind, false)
	cagg := counter.New()
	aggtest.CheckedUpdate(fix.t, cagg, core.NewInt64Number(123), desc)
	cagg.Checkpoint(fix.ctx, desc)

	checkpointSet.Add(desc, cagg, key.String("A", "B"), key.String("C", "D"))

	fix.Export(checkpointSet)

	require.Equal(t, `{"updates":[{"name":"test.name{A=B,C=D}","sum":123}]}`, fix.Output())
}

func TestStdoutLastValueFormat(t *testing.T) {
	fix := newFixture(t, stdout.Config{})

	checkpointSet := test.NewCheckpointSet(sdk.NewDefaultLabelEncoder())

	desc := export.NewDescriptor("test.name", export.ObserverKind, nil, "", "", core.Float64NumberKind, false)
	lvagg := lastvalue.New()
	aggtest.CheckedUpdate(fix.t, lvagg, core.NewFloat64Number(123.456), desc)
	lvagg.Checkpoint(fix.ctx, desc)

	checkpointSet.Add(desc, lvagg, key.String("A", "B"), key.String("C", "D"))

	fix.Export(checkpointSet)

	require.Equal(t, `{"updates":[{"name":"test.name{A=B,C=D}","last":123.456}]}`, fix.Output())
}

func TestStdoutMinMaxSumCount(t *testing.T) {
	fix := newFixture(t, stdout.Config{})

	checkpointSet := test.NewCheckpointSet(sdk.NewDefaultLabelEncoder())

	desc := export.NewDescriptor("test.name", export.MeasureKind, nil, "", "", core.Float64NumberKind, false)
	magg := minmaxsumcount.New(desc)
	aggtest.CheckedUpdate(fix.t, magg, core.NewFloat64Number(123.456), desc)
	aggtest.CheckedUpdate(fix.t, magg, core.NewFloat64Number(876.543), desc)
	magg.Checkpoint(fix.ctx, desc)

	checkpointSet.Add(desc, magg, key.String("A", "B"), key.String("C", "D"))

	fix.Export(checkpointSet)

	require.Equal(t, `{"updates":[{"name":"test.name{A=B,C=D}","min":123.456,"max":876.543,"sum":999.999,"count":2}]}`, fix.Output())
}

func TestStdoutMeasureFormat(t *testing.T) {
	fix := newFixture(t, stdout.Config{
		PrettyPrint: true,
	})

	checkpointSet := test.NewCheckpointSet(sdk.NewDefaultLabelEncoder())

	desc := export.NewDescriptor("test.name", export.MeasureKind, nil, "", "", core.Float64NumberKind, false)
	magg := array.New()

	for i := 0; i < 1000; i++ {
		aggtest.CheckedUpdate(fix.t, magg, core.NewFloat64Number(float64(i)+0.5), desc)
	}

	magg.Checkpoint(fix.ctx, desc)

	checkpointSet.Add(desc, magg, key.String("A", "B"), key.String("C", "D"))

	fix.Export(checkpointSet)

	require.Equal(t, `{
	"updates": [
		{
			"name": "test.name{A=B,C=D}",
			"min": 0.5,
			"max": 999.5,
			"sum": 500000,
			"count": 1000,
			"quantiles": [
				{
					"q": 0.5,
					"v": 500.5
				},
				{
					"q": 0.9,
					"v": 900.5
				},
				{
					"q": 0.99,
					"v": 990.5
				}
			]
		}
	]
}`, fix.Output())
}

func TestStdoutEmptyDataSet(t *testing.T) {
	desc := export.NewDescriptor("test.name", export.MeasureKind, nil, "", "", core.Float64NumberKind, false)
	for name, tc := range map[string]export.Aggregator{
		"ddsketch":       ddsketch.New(ddsketch.NewDefaultConfig(), desc),
		"minmaxsumcount": minmaxsumcount.New(desc),
	} {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fix := newFixture(t, stdout.Config{})

			checkpointSet := test.NewCheckpointSet(sdk.NewDefaultLabelEncoder())

			magg := tc
			magg.Checkpoint(fix.ctx, desc)

			checkpointSet.Add(desc, magg)

			fix.Export(checkpointSet)

			require.Equal(t, `{"updates":null}`, fix.Output())
		})
	}
}

func TestStdoutLastValueNotSet(t *testing.T) {
	fix := newFixture(t, stdout.Config{})

	checkpointSet := test.NewCheckpointSet(sdk.NewDefaultLabelEncoder())

	desc := export.NewDescriptor("test.name", export.ObserverKind, nil, "", "", core.Float64NumberKind, false)
	lvagg := lastvalue.New()
	lvagg.Checkpoint(fix.ctx, desc)

	checkpointSet.Add(desc, lvagg, key.String("A", "B"), key.String("C", "D"))

	fix.Export(checkpointSet)

	require.Equal(t, `{"updates":null}`, fix.Output())
}

func TestStdoutCounterWithUnspecifiedKeys(t *testing.T) {
	fix := newFixture(t, stdout.Config{})

	checkpointSet := test.NewCheckpointSet(sdk.NewDefaultLabelEncoder())

	keys := []core.Key{key.New("C"), key.New("D")}

	desc := export.NewDescriptor("test.name", export.CounterKind, keys, "", "", core.Int64NumberKind, false)
	cagg := counter.New()
	aggtest.CheckedUpdate(fix.t, cagg, core.NewInt64Number(10), desc)
	cagg.Checkpoint(fix.ctx, desc)

	checkpointSet.Add(desc, cagg, key.String("A", "B"))

	fix.Export(checkpointSet)

	require.Equal(t, `{"updates":[{"name":"test.name{A=B,C,D}","sum":10}]}`, fix.Output())
}
