package volumes

import (
	"fmt"
	"path"

	corev1 "k8s.io/api/core/v1"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
)

type VolumeInfo struct {
	Volume corev1.Volume
	Mount  corev1.VolumeMount
}

type Builder struct {
	info []VolumeInfo
}

func (b *Builder) AddMutualTLSCertificates(config *flowslatest.ClientTLS, namePrefix string) (caPath, userCertPath, userKeyPath string) {
	if config.CACert.Name != "" {
		caPath, _ = b.AddCertificate(&config.CACert, namePrefix+"-ca")
	}
	if config.UserCert.Name != "" {
		userCertPath, userKeyPath = b.AddCertificate(&config.UserCert, namePrefix+"-user")
	}
	return
}

func (b *Builder) AddCACertificate(config *flowslatest.ClientTLS, namePrefix string) (caPath string) {
	if config.CACert.Name != "" {
		caPath, _ = b.AddCertificate(&config.CACert, namePrefix+"-ca")
	}
	return
}

func (b *Builder) AddCertificate(ref *flowslatest.CertificateReference, volumeName string) (certPath, keyPath string) {
	if ref != nil && ref.Name != "" {
		certPath = fmt.Sprintf("/var/%s/%s", volumeName, ref.CertFile)
		keyPath = fmt.Sprintf("/var/%s/%s", volumeName, ref.CertKey)
		vol, vm := buildVolumeAndMount(ref.Type, ref.Name, volumeName)
		b.insertOrReplace(&VolumeInfo{Volume: vol, Mount: vm})
	}
	return
}

func (b *Builder) AddVolume(config *flowslatest.FileReference, volumeName string) string {
	vol, vm := buildVolumeAndMount(config.Type, config.Name, volumeName)
	b.insertOrReplace(&VolumeInfo{Volume: vol, Mount: vm})
	return path.Join("/var", volumeName, config.File)
}

// AddToken will add a volume + volume mount for a service account token if defined
func (b *Builder) AddToken(name string) string {
	for i := range b.info {
		if b.info[i].Volume.Name == name {
			// Token already added
			return constants.TokensPath + name
		}
	}
	vol := corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{
						ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
							Path: name,
						},
					},
				},
			},
		},
	}
	vm := corev1.VolumeMount{
		MountPath: constants.TokensPath,
		Name:      name,
	}
	b.info = append(b.info, VolumeInfo{Volume: vol, Mount: vm})
	return constants.TokensPath + name
}

func (b *Builder) GetVolumes() []corev1.Volume {
	var vols []corev1.Volume
	for i := range b.info {
		vols = append(vols, b.info[i].Volume)
	}
	return vols
}

func (b *Builder) GetMounts() []corev1.VolumeMount {
	var vols []corev1.VolumeMount
	for i := range b.info {
		vols = append(vols, b.info[i].Mount)
	}
	return vols
}

func (b *Builder) AppendVolumes(existing []corev1.Volume) []corev1.Volume {
	return append(existing, b.GetVolumes()...)
}

func (b *Builder) AppendMounts(existing []corev1.VolumeMount) []corev1.VolumeMount {
	return append(existing, b.GetMounts()...)
}

func buildVolumeAndMount(refType flowslatest.MountableType, refName string, volumeName string) (corev1.Volume, corev1.VolumeMount) {
	var vol corev1.Volume
	if refType == flowslatest.RefTypeConfigMap {
		vol = corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: refName,
					},
				},
			},
		}
	} else {
		vol = corev1.Volume{
			Name: volumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: refName,
				},
			},
		}
	}
	return vol, corev1.VolumeMount{
		Name:      volumeName,
		ReadOnly:  true,
		MountPath: "/var/" + volumeName,
	}
}

func (b *Builder) insertOrReplace(vi *VolumeInfo) {
	// find any existing volume info and replace it
	for i := range b.info {
		if b.info[i].Volume.Name == vi.Volume.Name || b.info[i].Mount.Name == vi.Mount.Name {
			b.info[i] = *vi
			return
		}
	}

	// else just append new volume info
	b.info = append(b.info, *vi)
}
