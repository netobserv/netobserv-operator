package ebpf

import (
	"strings"

	ebpfmaps "github.com/netobserv/netobserv-ebpf-agent/pkg/maps"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/pkg/volumes"
	corev1 "k8s.io/api/core/v1"
)

const (
	bpfTraceMountName               = "bpf-kernel-debug"
	bpfTraceMountPath               = "/sys/kernel/debug"
	bpfNetNSMountName               = "var-run-netns"
	bpfNetNSMountPath               = "/var/run/netns"
	ovnObservMountName              = "var-run-ovn"
	ovnObservMountPath              = "/var/run/ovn"
	ovnObservMountNameOpenShift     = "var-run-ovn-openshift"
	ovnObservHostMountPathOpenShift = "/var/run/ovn-ic"
	ovsMountPath                    = "/var/run/openvswitch"
	ovsHostMountPath                = "/var/run/openvswitch"
	ovsMountName                    = "var-run-ovs"
	bpfmanMapsVolumeName            = "bpfman-maps"
	bpfManBpfFSPath                 = "/run/netobserv/maps"
)

type volumeInfo struct {
	name        string
	envOverride string
	hostPath    string
	mountPath   string
}

func getHostVolumesInfo(spec *flowslatest.FlowCollectorEBPF, vendor constants.Vendor) []volumeInfo {
	var info []volumeInfo
	if spec.IsSecondaryNetworksEnabled() {
		info = append(info, volumeInfo{name: bpfNetNSMountName, hostPath: bpfNetNSMountPath, mountPath: bpfNetNSMountPath})
	}

	// EBPF Manager takes care of mounting the kernel debug volume.
	if spec.IsAgentFeatureEnabled(flowslatest.PacketDrop) && !spec.IsEbpfManagerEnabled() {
		info = append(info, volumeInfo{name: bpfTraceMountName, hostPath: bpfTraceMountPath, mountPath: bpfTraceMountPath})
	}

	if spec.IsNetworkEventsEnabled() || spec.IsUDNMappingEnabled() {
		hostPath := ovsHostMountPath
		if vendor == constants.VendorOpenShift || vendor == constants.VendorOpenShiftDownstream {
			hostPath = ovnObservHostMountPathOpenShift
		}
		info = append(info, volumeInfo{name: ovnObservMountName, hostPath: hostPath, mountPath: ovnObservMountPath, envOverride: envOVNObservHostMountPath})
		info = append(info, volumeInfo{name: ovsMountName, hostPath: ovsHostMountPath, mountPath: ovsMountPath})
	}
	return info
}

func buildVolumes(spec *flowslatest.FlowCollectorEBPF, vendor constants.Vendor, vols volumes.Builder, envOverrides map[string]string) ([]corev1.VolumeMount, []corev1.Volume) {
	volumeMounts := vols.GetMounts()
	volumes := vols.GetVolumes()

	for _, info := range getHostVolumesInfo(spec, vendor) {
		hostPath := info.hostPath
		if o := envOverrides[info.envOverride]; o != "" {
			hostPath = o
		}
		volume := corev1.Volume{
			Name: info.name,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Type: newHostPathType(corev1.HostPathDirectory),
					Path: hostPath,
				},
			},
		}
		volumes = append(volumes, volume)
		volumeMount := corev1.VolumeMount{
			Name:             info.name,
			MountPath:        info.mountPath,
			MountPropagation: newMountPropagationMode(corev1.MountPropagationHostToContainer),
		}
		volumeMounts = append(volumeMounts, volumeMount)
	}

	if spec.IsEbpfManagerEnabled() {
		volume := corev1.Volume{
			Name: bpfmanMapsVolumeName,
			VolumeSource: corev1.VolumeSource{
				CSI: &corev1.CSIVolumeSource{
					Driver: "csi.bpfman.io",
					VolumeAttributes: map[string]string{
						"csi.bpfman.io/program": "netobserv",
						"csi.bpfman.io/maps":    strings.Join(ebpfmaps.Maps, ","),
					},
				},
			},
		}
		volumes = append(volumes, volume)
		volumeMount := corev1.VolumeMount{
			Name:             bpfmanMapsVolumeName,
			MountPath:        bpfManBpfFSPath,
			MountPropagation: newMountPropagationMode(corev1.MountPropagationHostToContainer),
		}
		volumeMounts = append(volumeMounts, volumeMount)
	}

	return volumeMounts, volumes
}
