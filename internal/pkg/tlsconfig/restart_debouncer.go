package tlsconfig

import (
	"context"
	"reflect"
	"sync"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TLSProfileRestartDebouncer delays a restart until no TLS profile changes have been
// observed for the configured interval. The applied profile remains immutable because
// it represents the configuration used to start the metrics and webhook servers.
type TLSProfileRestartDebouncer struct {
	applied  configv1.TLSProfileSpec
	interval time.Duration
	restart  context.CancelFunc

	mu        sync.Mutex
	latest    configv1.TLSProfileSpec
	changedAt time.Time
	changed   chan struct{}
}

// NewTLSProfileRestartDebouncer creates a debouncer using the profile applied to the
// running servers as its immutable baseline.
func NewTLSProfileRestartDebouncer(
	applied configv1.TLSProfileSpec,
	interval time.Duration,
	restart context.CancelFunc,
) *TLSProfileRestartDebouncer {
	return &TLSProfileRestartDebouncer{
		applied:  *applied.DeepCopy(),
		interval: interval,
		restart:  restart,
		latest:   *applied.DeepCopy(),
		changed:  make(chan struct{}, 1),
	}
}

// Observe records the most recently observed profile and resets the quiet period.
func (d *TLSProfileRestartDebouncer) Observe(profile configv1.TLSProfileSpec) {
	d.mu.Lock()
	d.latest = *profile.DeepCopy()
	d.changedAt = time.Now()
	d.mu.Unlock()

	select {
	case d.changed <- struct{}{}:
	default:
	}
}

// Start waits for profile changes and requests a graceful restart once the latest
// profile has remained different from the applied profile for the quiet period.
func (d *TLSProfileRestartDebouncer) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	var timer *time.Timer
	var timerC <-chan time.Time
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-d.changed:
			if timer == nil {
				timer = time.NewTimer(d.interval)
			} else {
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(d.interval)
			}
			timerC = timer.C
		case <-timerC:
			d.mu.Lock()
			latest := *d.latest.DeepCopy()
			changedAt := d.changedAt

			// An observation can race with the timer. Ensure a complete quiet period
			// has elapsed since the latest one before deciding to restart.
			if remaining := d.interval - time.Since(changedAt); remaining > 0 {
				d.mu.Unlock()
				timer.Reset(remaining)
				timerC = timer.C
				continue
			}
			timerC = nil

			if reflect.DeepEqual(d.applied, latest) {
				d.mu.Unlock()
				logger.Info("TLS profile returned to the applied configuration; reload is not needed")
				continue
			}

			logger.Info("TLS profile has stabilized, initiating graceful shutdown to reload",
				"appliedProfile", d.applied, "newProfile", latest)
			d.restart()
			d.mu.Unlock()
			return nil
		}
	}
}

// NeedLeaderElection makes the debounce active on every operator replica, matching the
// TLS profile watcher which also runs without leader election.
func (d *TLSProfileRestartDebouncer) NeedLeaderElection() bool {
	return false
}
