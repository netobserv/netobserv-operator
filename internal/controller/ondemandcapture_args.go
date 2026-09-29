package controllers

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	odcapi "github.com/netobserv/netobserv-operator/api/ondemandcapture/v1beta1"
)

var captureFilterNames = strings.Fields("direction cidr protocol sport dport port sport_range dport_range port_range sports dports ports icmp_type icmp_code peer_ip peer_cidr action tcp_flags drops")
var captureValueOptions = strings.Fields("max-time max-bytes log-level sampling node-selector query include_list interfaces exclude_interfaces")
var captureBoolOptions = strings.Fields("privileged enable_all get-subnets drops enable_pkt_drop enable_dns enable_rtt enable_network_events enable_udn_mapping enable_pkt_translation enable_ipsec")
var managedCaptureOptions = strings.Fields("namespace kubeconfig context output-dir headless background yaml copy help version")

// Values remain separate argv entries: no shell interpretation or quoting is used.
// The CLI validates option values and mode-specific combinations before creating resources.
func validateCaptureArgs(args []string) (bool, error) {
	hasFilter := false
	for i := 0; i < len(args); i++ {
		if args[i] == "or" {
			continue
		}
		key, value, hasValue := strings.Cut(strings.TrimPrefix(args[i], "--"), "=")
		if slices.Contains(managedCaptureOptions, key) {
			return false, fmt.Errorf("--%s is managed by the operator and cannot be set in args", key)
		}
		isBool := slices.Contains(captureBoolOptions, key)
		isFilter := slices.Contains(captureFilterNames, key)
		if !isBool && !isFilter && !slices.Contains(captureValueOptions, key) {
			return false, fmt.Errorf("unsupported capture argument %q", args[i])
		}
		if !hasValue && !isBool {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				return false, fmt.Errorf("missing value for --%s", key)
			}
			i++
			value = args[i]
		}
		// The CLI treats any argument ending in help as a help request.
		if strings.HasSuffix(args[i], "help") {
			return false, fmt.Errorf("help requests cannot run as captures")
		}
		if isFilter && (key != "drops" || value != "false") {
			hasFilter = true
		}
	}
	return hasFilter, nil
}

// validateFilterGroups rejects empty alternative groups that would broaden a capture.
func validateFilterGroups(filters *odcapi.CaptureFilters) error {
	if filters != nil {
		for i := range filters.Or {
			if len(captureRuleArgs(&filters.Or[i])) == 0 {
				return fmt.Errorf("filters.or groups must contain at least one filter")
			}
		}
	}
	return nil
}

// captureFilterArgs preserves ordered OR groups when translating structured filters.
func captureFilterArgs(filters *odcapi.CaptureFilters) []string {
	if filters == nil {
		return nil
	}
	args := captureRuleArgs(&filters.CaptureFilterRule)
	for i := range filters.Or {
		group := captureRuleArgs(&filters.Or[i])
		if len(args) > 0 {
			args = append(args, "or")
		}
		args = append(args, group...)
	}
	return args
}

// captureRuleArgs serializes a single filter group using native CLI option names.
func captureRuleArgs(rule *odcapi.CaptureFilterRule) []string {
	args := []string{}
	values := []struct{ key, value string }{
		{"direction", rule.Direction}, {"cidr", rule.CIDR}, {"protocol", rule.Protocol},
		{"sport", filterInt(rule.SPort)}, {"dport", filterInt(rule.DPort)}, {"port", filterInt(rule.Port)},
		{"sport_range", rule.SPortRange}, {"dport_range", rule.DPortRange}, {"port_range", rule.PortRange},
		{"sports", rule.Sports}, {"dports", rule.Dports}, {"ports", rule.Ports},
		{"icmp_type", filterInt(rule.ICMPType)}, {"icmp_code", filterInt(rule.ICMPCode)},
		{"peer_ip", rule.PeerIP}, {"peer_cidr", rule.PeerCIDR}, {"action", rule.Action}, {"tcp_flags", rule.TCPFlags},
	}
	for _, v := range values {
		if v.value != "" {
			args = append(args, "--"+v.key+"="+v.value)
		}
	}
	if rule.Drops {
		args = append(args, "--drops")
	}
	return args
}

// filterInt distinguishes an omitted numeric filter from an explicitly requested zero.
func filterInt(value *int32) string {
	if value == nil {
		return ""
	}
	return strconv.FormatInt(int64(*value), 10)
}
