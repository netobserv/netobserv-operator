# OnDemandCapture

The current API behavior, supported filters, examples, authenticated download
workflow, and retention policy are documented in
[OnDemandCapture with the Go CLI](OnDemandCapture-Go.md).

The former reference described a different backend. In particular, namespace
and pod-label filters are not supported, the file server is reachable through
Kubernetes port forwarding rather than the pod IP, and metrics remain in
Prometheus rather than a downloadable archive.
