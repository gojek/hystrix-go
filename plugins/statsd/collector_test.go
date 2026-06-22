package statsd

import (
	"testing"
)

func TestSampleRate(t *testing.T) {
	t.Parallel()
	t.Run(`with no sample rate`, func(t *testing.T) {
		t.Parallel()

		client, err := InitializeCollector(&CollectorConfig{
			StatsdAddr: "localhost:8125",
			Prefix:     "test",
		})
		if err != nil {
			t.Fatalf("error initializing statsd collector: %v", err)
		}

		collector := client.NewStatsdCollector("foo").(*Collector)
		if collector.sampleRate != 1.0 {
			t.Errorf("sampleRate = %f, want %f", collector.sampleRate, 1.0)
		}
	})

	t.Run(`with sample rate`, func(t *testing.T) {
		t.Parallel()

		client, err := InitializeCollector(&CollectorConfig{
			StatsdAddr: "localhost:8125",
			Prefix:     "test",
			SampleRate: 0.5,
		})
		if err != nil {
			t.Fatalf("error initializing statsd collector: %v", err)
		}

		collector := client.NewStatsdCollector("foo").(*Collector)
		if collector.sampleRate != 0.5 {
			t.Errorf("sampleRate = %f, want %f", collector.sampleRate, 0.5)
		}
	})
}
