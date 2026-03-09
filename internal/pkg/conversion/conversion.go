// Package conversion implements conversion utilities.
package conversion

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
)

// Following K8S convention, mixed capitalization should be preserved
// see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#constants
var upperPascalExceptions = map[string]string{
	"IPFIX":        "IPFIX",
	"EBPF":         "eBPF",
	"SCRAM-SHA512": "ScramSHA512",
}

var upperTokenizer = regexp.MustCompile(`[\-\_]+`)

// MarshalData stores the source object as json data in the destination object annotations map.
// It ignores the metadata of the source object.
func MarshalData(src metav1.Object, dst metav1.Object) error {
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(src)
	if err != nil {
		return err
	}
	delete(u, "metadata")

	data, err := json.Marshal(u)
	if err != nil {
		return err
	}
	annotations := dst.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations[constants.ConversionAnnotation] = string(data)
	dst.SetAnnotations(annotations)
	return nil
}

// UnmarshalData tries to retrieve the data from the annotation and unmarshals it into the object passed as input.
func UnmarshalData(from metav1.Object, to interface{}) (bool, error) {
	annotations := from.GetAnnotations()
	data, ok := annotations[constants.ConversionAnnotation]
	if !ok {
		return false, nil
	}
	if err := json.Unmarshal([]byte(data), to); err != nil {
		return false, err
	}
	delete(annotations, constants.ConversionAnnotation)
	from.SetAnnotations(annotations)
	return true, nil
}

func UpperToPascal(str string) string {
	if len(str) == 0 {
		return str
	}

	// check for any exception in map
	if exception, found := upperPascalExceptions[str]; found {
		return exception
	}

	// Split on '-' or '_' rune, capitalize first letter of each part and join them
	var sb strings.Builder
	array := upperTokenizer.Split(strings.ToLower(str), -1)
	for _, s := range array {
		runes := []rune(s)
		runes[0] = unicode.ToUpper(runes[0])
		sb.WriteString(string(runes))
	}
	return sb.String()
}

func PascalToUpper(str string, splitter rune) string {
	if len(str) == 0 {
		return str
	}

	// check for any exception in map
	for k, v := range upperPascalExceptions {
		if v == str {
			return k
		}
	}

	// Split on capital letters, upper each part and join with splitter
	var sb strings.Builder
	runes := []rune(str)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			sb.WriteRune(splitter)
		}
		sb.WriteRune(unicode.ToUpper(r))
	}
	return sb.String()
}

func PascalToLower(str string, splitter rune) string {
	if len(str) == 0 {
		return str
	}

	// check for any exception in map
	for k, v := range upperPascalExceptions {
		if v == str {
			return k
		}
	}

	// Split on capital letters, upper each part and join with splitter
	var sb strings.Builder
	runes := []rune(str)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			sb.WriteRune(splitter)
		}
		sb.WriteRune(unicode.ToLower(r))
	}
	return sb.String()
}
