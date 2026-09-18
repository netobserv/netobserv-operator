package tlsconfig

import (
	"context"
	"testing"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	configv1 "github.com/openshift/api/config/v1"
)

func TestTLSProfileRestartDebouncer(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "TLS profile restart debouncer")
}

var _ = ginkgo.Describe("TLS profile restart debouncer", func() {
	var (
		ctx     context.Context
		cancel  context.CancelFunc
		stopped chan struct{}
		done    chan error
	)

	start := func(applied configv1.TLSProfileSpec, interval time.Duration) *TLSProfileRestartDebouncer {
		ctx, cancel = context.WithCancel(context.Background())
		stopped = make(chan struct{}, 1)
		done = make(chan error, 1)
		debouncer := NewTLSProfileRestartDebouncer(applied, interval, func() {
			stopped <- struct{}{}
		})
		go func() {
			done <- debouncer.Start(ctx)
		}()
		return debouncer
	}

	ginkgo.AfterEach(func() {
		if cancel != nil {
			cancel()
		}
		gomega.Eventually(done).Should(gomega.Receive(gomega.BeNil()))
	})

	ginkgo.It("restarts only after a changed profile remains stable", func() {
		applied := *configv1.TLSProfiles[configv1.TLSProfileIntermediateType]
		modern := *configv1.TLSProfiles[configv1.TLSProfileModernType]
		debouncer := start(applied, 40*time.Millisecond)

		debouncer.Observe(modern)
		gomega.Consistently(stopped, 20*time.Millisecond).ShouldNot(gomega.Receive())
		gomega.Eventually(stopped).Should(gomega.Receive())
	})

	ginkgo.It("resets the quiet period when another profile is observed", func() {
		applied := *configv1.TLSProfiles[configv1.TLSProfileIntermediateType]
		old := *configv1.TLSProfiles[configv1.TLSProfileOldType]
		modern := *configv1.TLSProfiles[configv1.TLSProfileModernType]
		debouncer := start(applied, 50*time.Millisecond)

		debouncer.Observe(old)
		gomega.Consistently(stopped, 30*time.Millisecond).ShouldNot(gomega.Receive())
		debouncer.Observe(modern)
		gomega.Consistently(stopped, 30*time.Millisecond).ShouldNot(gomega.Receive())
		gomega.Eventually(stopped).Should(gomega.Receive())
	})

	ginkgo.It("does not restart when the profile returns to the applied value", func() {
		applied := *configv1.TLSProfiles[configv1.TLSProfileIntermediateType]
		modern := *configv1.TLSProfiles[configv1.TLSProfileModernType]
		debouncer := start(applied, 30*time.Millisecond)

		debouncer.Observe(modern)
		debouncer.Observe(applied)
		gomega.Consistently(stopped, 60*time.Millisecond).ShouldNot(gomega.Receive())
	})

	ginkgo.It("stops cleanly when the manager context is cancelled", func() {
		applied := *configv1.TLSProfiles[configv1.TLSProfileIntermediateType]
		modern := *configv1.TLSProfiles[configv1.TLSProfileModernType]
		debouncer := start(applied, time.Hour)

		debouncer.Observe(modern)
		cancel()
		gomega.Consistently(stopped, 20*time.Millisecond).ShouldNot(gomega.Receive())
	})
})
