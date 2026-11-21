package api

// IngestHAProxy receives versioned HAProxy access logs over UDP syslog.
type IngestHAProxy struct {
	Address        string `yaml:"address,omitempty" json:"address,omitempty" doc:"UDP listen address (default: :10514)"`
	BufferLen      int    `yaml:"bufferLength,omitempty" json:"bufferLength,omitempty" doc:"maximum queued syslog datagrams (default: 1000)"`
	MaxMessageSize int    `yaml:"maxMessageSize,omitempty" json:"maxMessageSize,omitempty" doc:"maximum datagram size in bytes, from 512 to 65507 (default: 2048)"`
}
