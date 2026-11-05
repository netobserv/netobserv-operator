package v1alpha1

import "testing"

func TestFlowCaptureWarnings(t *testing.T) {
	tests := []struct {
		name     string
		capture  FlowCapture
		warnings int
	}{
		{name: "unbounded capture", capture: FlowCapture{Spec: FlowCaptureSpec{OutputType: Flows}}, warnings: 1},
		{name: "maxBytes is ignored for metrics", capture: FlowCapture{Spec: FlowCaptureSpec{OutputType: Metrics, MaxBytes: 1}}, warnings: 1},
		{name: "maxBytes applies to flows", capture: FlowCapture{Spec: FlowCaptureSpec{OutputType: Flows, MaxBytes: 1}}, warnings: 0},
		{name: "maxBytes applies to packets", capture: FlowCapture{Spec: FlowCaptureSpec{OutputType: Packets, MaxBytes: 1}}, warnings: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(flowCaptureWarnings(&tt.capture)); got != tt.warnings {
				t.Fatalf("flowCaptureWarnings() returned %d warnings, want %d", got, tt.warnings)
			}
		})
	}
}

func TestValidateFlowCaptureTLSPlaintext(t *testing.T) {
	capture := FlowCapture{Spec: FlowCaptureSpec{OutputType: Flows, TLSPlaintext: &TLSPlaintextSpec{}}}
	if err := validateFlowCapture(&capture); err == nil {
		t.Fatal("expected TLS plaintext to be rejected for flow captures")
	}
	capture.Spec.OutputType = Packets
	if err := validateFlowCapture(&capture); err != nil {
		t.Fatalf("expected TLS plaintext packet capture to be accepted: %v", err)
	}
}
