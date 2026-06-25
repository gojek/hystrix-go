package hystrix

import (
	"cmp"
	"maps"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// DefaultTimeout is how long to wait for command to complete, in milliseconds
	DefaultTimeout = 1000
	// DefaultMaxConcurrent is how many commands of the same type can run at the same time
	DefaultMaxConcurrent = 10
	// DefaultVolumeThreshold is the minimum number of requests needed before a circuit can be tripped due to health
	DefaultVolumeThreshold = 20
	// DefaultSleepWindow is how long, in milliseconds, to wait after a circuit opens before testing for recovery
	DefaultSleepWindow = 5000
	// DefaultErrorPercentThreshold causes circuits to open once the rolling measure of errors exceeds this percent of requests
	DefaultErrorPercentThreshold = 50
	// DefaultLogger is the default logger that will be used in the Hystrix package. By default prints nothing.
	DefaultLogger = NoopLogger{}
)

type Settings struct {
	Timeout                time.Duration
	MaxConcurrentRequests  int
	RequestVolumeThreshold uint64
	SleepWindow            time.Duration
	ErrorPercentThreshold  int
}

// CommandConfig is used to tune circuit settings at runtime
type CommandConfig struct {
	Timeout                int `json:"timeout"`
	MaxConcurrentRequests  int `json:"max_concurrent_requests"`
	RequestVolumeThreshold int `json:"request_volume_threshold"`
	SleepWindow            int `json:"sleep_window"`
	ErrorPercentThreshold  int `json:"error_percent_threshold"`
}

// settingsMutex only used for adding new settings to limit concurrent writes. Each writes involve copying existing
// circuitSettings, modifying it and then doing atomic store operation. This setup allows us to skip
// sync.Mutex/sync.RWMutex operations for happy path reads, which is the most common case
var settingsMutex sync.Mutex
var circuitSettings atomic.Pointer[map[string]*Settings]
var log logger

func init() {
	circuitSettings.Store(&map[string]*Settings{})
	log = DefaultLogger
}

// Configure applies settings for a set of circuits
func Configure(cmds map[string]CommandConfig) {
	settingsMutex.Lock() // mutex to ensure only one configure happens at a time
	defer settingsMutex.Unlock()

	settings := maps.Clone(*circuitSettings.Load()) // clone so that all concurrent operation are limited to read
	for k, v := range cmds {
		settings[k] = &Settings{
			Timeout:                time.Duration(cmp.Or(v.Timeout, DefaultTimeout)) * time.Millisecond,
			MaxConcurrentRequests:  cmp.Or(v.MaxConcurrentRequests, DefaultMaxConcurrent),
			RequestVolumeThreshold: uint64(cmp.Or(v.RequestVolumeThreshold, DefaultVolumeThreshold)),
			SleepWindow:            time.Duration(cmp.Or(v.SleepWindow, DefaultSleepWindow)) * time.Millisecond,
			ErrorPercentThreshold:  cmp.Or(v.ErrorPercentThreshold, DefaultErrorPercentThreshold),
		}
	}
	circuitSettings.Store(&settings)
}

// ConfigureCommand applies settings for a circuit
func ConfigureCommand(name string, config CommandConfig) {
	Configure(map[string]CommandConfig{name: config})
}

func getSettings(name string) *Settings {
	s, exists := (*circuitSettings.Load())[name]

	if !exists {
		ConfigureCommand(name, CommandConfig{})
		s = getSettings(name)
	}

	return s
}

func GetCircuitSettings() map[string]*Settings {
	return maps.Clone(*circuitSettings.Load())
}

// SetLogger configures the logger that will be used. This only applies to the hystrix package.
func SetLogger(l logger) {
	log = l
}
