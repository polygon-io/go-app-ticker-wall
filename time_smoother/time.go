package time_smoother

import (
	"math"
	"time"

	"github.com/dterei/gotsc"
)

type TimeSmoother struct {
	lastRdtsc      uint64 // last cycle count
	lastRdtscNanos uint64 // last cycle-synced time
	lastSystemTime uint64 // last system time
	rdtscTime      uint64 // time to call rdtsc

	clockSpeed          float64
	clockSpeedDecayRate float64

	adjustedTimeRate float64 // nanos per clock

	syncInterval uint64

	stop chan bool
}

func (t *TimeSmoother) Stop() {
	t.stop <- true
}

func (t *TimeSmoother) GetTime() time.Time {
	tnow := gotsc.BenchStart()

	// delta in cycles
	tDelta := float64(tnow - t.lastRdtsc)

	nanosNow := int64(tDelta*t.adjustedTimeRate) + int64(t.lastRdtscNanos)

	return time.Unix(0, nanosNow)
}

//
func (t *TimeSmoother) sync(systemTime, rdtsc uint64) {
	elapsedCycles := rdtsc - t.lastRdtsc
	elapsedNanos := int64(systemTime) - int64(t.lastSystemTime)

	nanosNow := int64(float64(elapsedCycles)*t.adjustedTimeRate) + int64(t.lastRdtscNanos) // current extrapolated time

	newClock := float64(elapsedNanos) / float64(elapsedCycles)
	t.clockSpeed = t.clockSpeedDecayRate*t.clockSpeed + (1-t.clockSpeedDecayRate)*newClock

	targetSystemTime := int64(systemTime + t.syncInterval)
	timeRate := float64(targetSystemTime-nanosNow) / t.clockSpeed
	if timeRate < 0 {
		timeRate = 0.0
	}

	t.adjustedTimeRate = timeRate
	t.lastSystemTime = systemTime
	t.lastRdtsc = rdtsc
	t.lastRdtscNanos = uint64(nanosNow)
}

func (t *TimeSmoother) daemon() {
	ticker := time.NewTicker(time.Duration(t.syncInterval))
	for {
		select {
		case <-ticker.C:
			systemTime := time.Now().UnixNano()
			rdtsc := gotsc.BenchStart() - t.rdtscTime
			t.sync(uint64(systemTime), rdtsc)
		case <-t.stop:
			return
		}
	}
}

// gets an initial estimate of the system clock speed
func (t *TimeSmoother) initClock() {
	sys1 := time.Now()
	rdtsc1 := gotsc.BenchStart()
	time.Sleep(time.Second)
	rdtsc2 := gotsc.BenchEnd()
	sysDelta := time.Since(sys1)

	cycles := rdtsc2 - rdtsc1 - 2*t.rdtscTime
	elapsedTime := sysDelta.Nanoseconds()

	clockSpeed := float64(elapsedTime) / float64(cycles)
	t.clockSpeed = clockSpeed
}

func NewTimeSmoother() *TimeSmoother {
	smoother := TimeSmoother{}
	smoother.rdtscTime = gotsc.TSCOverhead() / 2 // this function measures calling twice
	smoother.lastSystemTime = uint64(time.Now().UnixNano())
	smoother.lastRdtsc = gotsc.BenchStart() - smoother.rdtscTime
	smoother.lastRdtscNanos = smoother.lastSystemTime

	smoother.initClock()
	smoother.adjustedTimeRate = smoother.clockSpeed

	half_life := 10.0 // 10 second half life
	smoother.clockSpeedDecayRate = math.Pow(2, -1.0/half_life)
	smoother.syncInterval = 100_000_000 // 100 ms

	smoother.stop = make(chan bool)

	go smoother.daemon()

	return &smoother
}
