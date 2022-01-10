package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const MicrovmDatacenterKind = "MicrovmDatacenterConfig"

// Used for generating yaml for generate clusterconfig command
func NewMicrovmDatacenterConfigGenerate(clusterName string) *MicrovmDatacenterConfigGenerate {
	return &MicrovmDatacenterConfigGenerate{
		TypeMeta: metav1.TypeMeta{
			Kind:       MicrovmDatacenterKind,
			APIVersion: SchemeBuilder.GroupVersion.String(),
		},
		ObjectMeta: ObjectMeta{
			Name: clusterName,
		},
		Spec: MicrovmDatacenterConfigSpec{},
	}
}

func (c *MicrovmDatacenterConfigGenerate) APIVersion() string {
	return c.TypeMeta.APIVersion
}

func (c *MicrovmDatacenterConfigGenerate) Kind() string {
	return c.TypeMeta.Kind
}

func (c *MicrovmDatacenterConfigGenerate) Name() string {
	return c.ObjectMeta.Name
}

func GetMicrovmDatacenterConfig(fileName string) (*MicrovmDatacenterConfig, error) {
	var clusterConfig MicrovmDatacenterConfig
	err := ParseClusterConfig(fileName, &clusterConfig)
	if err != nil {
		return nil, err
	}
	return &clusterConfig, nil
}
