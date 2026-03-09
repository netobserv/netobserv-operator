// Package helpers contains some tools that are not related with any specific domain but required
// to perform some basic computational operations
package helper

import (
	"regexp"
	"sort"
	"strings"

	"github.com/netobserv/netobserv-operator/internal/controller/consoleplugin/config"
	"sigs.k8s.io/controller-runtime/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// maximum length of a metadata label in Kubernetes
const maxLabelLength = 63

func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func Intersect(first, second []string) bool {
	for _, i := range first {
		for _, j := range second {
			if i == j {
				return true
			}
		}
	}
	return false
}

func RemoveAllStrings(slice []string, search string) []string {
	for i, v := range slice {
		if v == search {
			return RemoveAllStrings(append(slice[:i], slice[i+1:]...), search)
		}
	}
	return slice
}

func ExtractVersion(image string) string {
	parts := strings.Split(image, ":")
	nparts := len(parts)
	if nparts > 1 {
		return parts[nparts-1]
	}
	return "unknown"
}

// IsSubSet returns whether the first argument contains all the keys and values of the second
// argument
func IsSubSet(set, subset map[string]string) bool {
	for k, v := range subset {
		if sv, ok := set[k]; !ok || v != sv {
			return false
		}
	}
	return true
}

// KeySorted returns the map key-value pairs sorted by Key
func KeySorted(set map[string]string) [][2]string {
	vals := make([][2]string, 0, len(set))
	for k, v := range set {
		vals = append(vals, [2]string{k, v})
	}
	sort.Slice(vals, func(i, j int) bool {
		return vals[i][0] < vals[j][0]
	})
	return vals
}

// MaxLabelLength cuts an input string it ifs length is largen than 63, the maximum length allowed
// by Kubernetes metadata
func MaxLabelLength(in string) string {
	if len(in) <= maxLabelLength {
		return in
	}
	return in[:maxLabelLength]
}

func UnstructuredDuration(in *metav1.Duration) string {
	if in == nil {
		return ""
	}
	return in.ToUnstructured().(string)
}

func FindFields(labels []string, isNumber bool) bool {
	type filter struct {
		exists bool
		isNum  bool
	}

	cfg, err := config.GetStaticFrontendConfig()
	if err != nil {
		return false
	}

	labelMap := make(map[string]filter)

	for _, f := range cfg.Fields {
		labelMap[f.Name] = filter{true, false}
		if f.Type == "number" {
			labelMap[f.Name] = filter{true, true}
		}
	}

	for _, l := range labels {
		// Split field for nesting, e.g. "NetworkEvents>Name" (and we don't verify the nested part)
		parts := strings.Split(l, ">")
		l = parts[0]
		if ok := labelMap[l].exists; !ok {
			return false
		}
		if isNumber && !labelMap[l].isNum {
			return false
		}
	}

	return true
}

func NamespacedName(obj client.Object) types.NamespacedName {
	return types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}
}

var promInvalidChars = regexp.MustCompile(`[^a-zA-Z0-9:_]`)

func PrometheusMetricName(from string) string {
	return promInvalidChars.ReplaceAllString(from, "_")
}
