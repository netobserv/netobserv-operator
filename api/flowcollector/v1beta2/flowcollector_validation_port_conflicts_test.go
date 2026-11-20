package v1beta2

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestPortConflictValidation(t *testing.T) {
	tests := []struct {
		name          string
		spec          FlowCollectorSpec
		expectError   bool
		errorContains string
	}{
		{
			name: "FLP port conflicts with k8scache port when informers enabled",
			spec: FlowCollectorSpec{
				Processor: FlowCollectorFLP{
					Advanced: &AdvancedProcessorConfig{
						Port: ptr.To(int32(DefaultK8sCachePort)),
					},
					InformerCacheProxy: &FlowCollectorInformerCacheProxy{
						Enabled: ptr.To(true),
					},
				},
			},
			expectError:   true,
			errorContains: "spec.processor.advanced.port 9402 conflicts with reserved k8scache port 9402",
		},
		{
			name: "Health port conflicts with k8scache port when informers enabled",
			spec: FlowCollectorSpec{
				Processor: FlowCollectorFLP{
					Advanced: &AdvancedProcessorConfig{
						HealthPort: ptr.To(int32(DefaultK8sCachePort)),
					},
					InformerCacheProxy: &FlowCollectorInformerCacheProxy{
						Enabled: ptr.To(true),
					},
				},
			},
			expectError:   true,
			errorContains: "spec.processor.advanced.healthPort 9402 conflicts with reserved k8scache port 9402",
		},
		{
			name: "Metrics port conflicts with k8scache port when informers enabled",
			spec: FlowCollectorSpec{
				Processor: FlowCollectorFLP{
					Metrics: FLPMetrics{
						Server: MetricsServerConfig{
							Port: ptr.To(int32(DefaultK8sCachePort)),
						},
					},
					InformerCacheProxy: &FlowCollectorInformerCacheProxy{
						Enabled: ptr.To(true),
					},
				},
			},
			expectError:   true,
			errorContains: "spec.processor.metrics.server.port 9402 conflicts with reserved k8scache port 9402",
		},
		{
			name: "Profile port conflicts with k8scache port when informers enabled",
			spec: FlowCollectorSpec{
				Processor: FlowCollectorFLP{
					Advanced: &AdvancedProcessorConfig{
						ProfilePort: ptr.To(int32(DefaultK8sCachePort)),
					},
					InformerCacheProxy: &FlowCollectorInformerCacheProxy{
						Enabled: ptr.To(true),
					},
				},
			},
			expectError:   true,
			errorContains: "spec.processor.advanced.profilePort 9402 conflicts with reserved k8scache port 9402",
		},
		{
			name: "Port DefaultK8sCachePort is allowed when informers disabled",
			spec: FlowCollectorSpec{
				Processor: FlowCollectorFLP{
					Advanced: &AdvancedProcessorConfig{
						Port: ptr.To(int32(DefaultK8sCachePort)),
					},
					InformerCacheProxy: &FlowCollectorInformerCacheProxy{
						Enabled: ptr.To(false),
					},
				},
			},
			expectError: false,
		},
		{
			name: "FLP port conflicts with default k8scache port when informers config is omitted",
			spec: FlowCollectorSpec{
				Processor: FlowCollectorFLP{
					Advanced: &AdvancedProcessorConfig{
						Port: ptr.To(int32(DefaultK8sCachePort)),
					},
					InformerCacheProxy: nil,
				},
			},
			expectError:   true,
			errorContains: "spec.processor.advanced.port 9402 conflicts with reserved k8scache port 9402",
		},
		{
			name: "Valid configuration with no port conflicts",
			spec: FlowCollectorSpec{
				Processor: FlowCollectorFLP{
					Advanced: &AdvancedProcessorConfig{
						Port:        ptr.To(int32(2055)),
						HealthPort:  ptr.To(int32(8080)),
						ProfilePort: ptr.To(int32(6060)),
					},
					Metrics: FLPMetrics{
						Server: MetricsServerConfig{
							Port: ptr.To(int32(9102)),
						},
					},
					InformerCacheProxy: &FlowCollectorInformerCacheProxy{
						Enabled: ptr.To(true),
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &FlowCollector{
				Spec: tt.spec,
			}

			_, err := fc.Validate(context.Background(), fc)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
