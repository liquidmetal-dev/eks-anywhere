package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MicrovmDatacenterConfigSpec defines the desired state of MicrovmDatacenterConfig.
type MicrovmDatacenterConfigSpec struct {
	FlintlockURL string `json:"flintlockURL"`
	MicrovmProxy string `json:"microvmProxy,omitempty"`
	SSHKey       string `json:"sshKey,omitempty"`
}

// MicrovmDatacenterConfigStatus defines the observed status for MicrovmDatacenterConfig.
type MicrovmDatacenterConfigStatus struct {
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// // MicrovmDatacenterConfig is the Schema for the MicrovmDatacenterConfigs API
type MicrovmDatacenterConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MicrovmDatacenterConfigSpec   `json:"spec,omitempty"`
	Status MicrovmDatacenterConfigStatus `json:"status,omitempty"`
}

func (t *MicrovmDatacenterConfig) Kind() string {
	return t.TypeMeta.Kind
}

func (v *MicrovmDatacenterConfig) ExpectedKind() string {
	return MicrovmDatacenterKind
}

func (v *MicrovmDatacenterConfig) PauseReconcile() {
	if v.Annotations == nil {
		v.Annotations = map[string]string{}
	}
	v.Annotations[pausedAnnotation] = "true"
}

func (v *MicrovmDatacenterConfig) IsReconcilePaused() bool {
	if s, ok := v.Annotations[pausedAnnotation]; ok {
		return s == "true"
	}
	return false
}

func (v *MicrovmDatacenterConfig) ClearPauseAnnotation() {
	if v.Annotations != nil {
		delete(v.Annotations, pausedAnnotation)
	}
}

func (v *MicrovmDatacenterConfig) ConvertConfigToConfigGenerateStruct() *MicrovmDatacenterConfigGenerate {
	namespace := defaultEksaNamespace
	if v.Namespace != "" {
		namespace = v.Namespace
	}
	config := &MicrovmDatacenterConfigGenerate{
		TypeMeta: v.TypeMeta,
		ObjectMeta: ObjectMeta{
			Name:        v.Name,
			Annotations: v.Annotations,
			Namespace:   namespace,
		},
		Spec: v.Spec,
	}

	return config
}

func (v *MicrovmDatacenterConfig) Marshallable() Marshallable {
	return v.ConvertConfigToConfigGenerateStruct()
}

// +kubebuilder:object:generate=false

// Same as MicrovmDatacenterConfig except stripped down for generation of yaml file during generate clusterconfig
type MicrovmDatacenterConfigGenerate struct {
	metav1.TypeMeta `json:",inline"`
	ObjectMeta      `json:"metadata,omitempty"`

	Spec MicrovmDatacenterConfigSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// MicrovmDatacenterConfigList contains a list of MicrovmDatacenterConfig
type MicrovmDatacenterConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MicrovmDatacenterConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MicrovmDatacenterConfig{}, &MicrovmDatacenterConfigList{})
}
