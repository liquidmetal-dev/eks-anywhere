package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MicrovmMachineConfigSpec defines the desired state of MicrovmMachineConfig
type MicrovmMachineConfigSpec struct {
	//TODO: add here
	OSFamily OSFamily            `json:"osFamily"`
	Users    []UserConfiguration `json:"users,omitempty"`
}

func (c *MicrovmMachineConfig) PauseReconcile() {
	c.Annotations[pausedAnnotation] = "true"
}

func (c *MicrovmMachineConfig) IsReconcilePaused() bool {
	if s, ok := c.Annotations[pausedAnnotation]; ok {
		return s == "true"
	}
	return false
}

func (c *MicrovmMachineConfig) SetControlPlane() {
	c.Annotations[controlPlaneAnnotation] = "true"
}

func (c *MicrovmMachineConfig) IsControlPlane() bool {
	if s, ok := c.Annotations[controlPlaneAnnotation]; ok {
		return s == "true"
	}
	return false
}

func (c *MicrovmMachineConfig) SetEtcd() {
	c.Annotations[etcdAnnotation] = "true"
}

func (c *MicrovmMachineConfig) IsEtcd() bool {
	if s, ok := c.Annotations[etcdAnnotation]; ok {
		return s == "true"
	}
	return false
}

func (c *MicrovmMachineConfig) SetManagement(clusterName string) {
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[managementAnnotation] = clusterName
}

func (c *MicrovmMachineConfig) IsManagement() bool {
	if s, ok := c.Annotations[managementAnnotation]; ok {
		return s != ""
	}
	return false
}

func (c *MicrovmMachineConfig) OSFamily() OSFamily {
	return c.Spec.OSFamily
}

func (c *MicrovmMachineConfig) GetNamespace() string {
	return c.Namespace
}

func (c *MicrovmMachineConfig) GetName() string {
	return c.Name
}

// MicrovmMachineConfigStatus defines the observed state of MicrovmMachineConfig
type MicrovmMachineConfigStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MicrovmMachineConfig is the Schema for the microvmmachineconfigs API
type MicrovmMachineConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MicrovmMachineConfigSpec   `json:"spec,omitempty"`
	Status MicrovmMachineConfigStatus `json:"status,omitempty"`
}

func (c *MicrovmMachineConfig) ConvertConfigToConfigGenerateStruct() *MicrovmMachineConfigGenerate {
	namespace := defaultEksaNamespace
	if c.Namespace != "" {
		namespace = c.Namespace
	}
	config := &MicrovmMachineConfigGenerate{
		TypeMeta: c.TypeMeta,
		ObjectMeta: ObjectMeta{
			Name:        c.Name,
			Annotations: c.Annotations,
			Namespace:   namespace,
		},
		Spec: c.Spec,
	}

	return config
}

func (c *MicrovmMachineConfig) Marshallable() Marshallable {
	return c.ConvertConfigToConfigGenerateStruct()
}

// +kubebuilder:object:generate=false

// Same as MicrovmMachineConfig except stripped down for generation of yaml file during generate clusterconfig
type MicrovmMachineConfigGenerate struct {
	metav1.TypeMeta `json:",inline"`
	ObjectMeta      `json:"metadata,omitempty"`

	Spec MicrovmMachineConfigSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// MicrovmMachineConfigList contains a list of MicrovmMachineConfig
type MicrovmMachineConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MicrovmMachineConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MicrovmMachineConfig{}, &MicrovmMachineConfigList{})
}
