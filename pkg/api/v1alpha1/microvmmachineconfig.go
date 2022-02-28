package v1alpha1

import (
	"fmt"
	"io/ioutil"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const MicrovmMachineConfigKind = "MicrovmMachineConfig"

// Used for generating yaml for generate clusterconfig command
func NewMicrovmMachineConfigGenerate(name string) *MicrovmMachineConfigGenerate {
	return &MicrovmMachineConfigGenerate{
		TypeMeta: metav1.TypeMeta{
			Kind:       MicrovmMachineConfigKind,
			APIVersion: SchemeBuilder.GroupVersion.String(),
		},
		ObjectMeta: ObjectMeta{
			Name: name,
		},
		Spec: MicrovmMachineConfigSpec{
			OSFamily: Ubuntu,
			Users: []UserConfiguration{{
				Name:              "ubuntu",
				SshAuthorizedKeys: []string{"ssh-rsa AAAA..."},
			}},
		},
	}
}

func (c *MicrovmMachineConfigGenerate) APIVersion() string {
	return c.TypeMeta.APIVersion
}

func (c *MicrovmMachineConfigGenerate) Kind() string {
	return c.TypeMeta.Kind
}

func (c *MicrovmMachineConfigGenerate) Name() string {
	return c.ObjectMeta.Name
}

func GetMicrovmMachineConfigs(fileName string) (map[string]*MicrovmMachineConfig, error) {
	configs := make(map[string]*MicrovmMachineConfig)
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("unable to read file due to: %v", err)
	}
	for _, c := range strings.Split(string(content), YamlSeparator) {
		var config MicrovmMachineConfig
		if err = yaml.UnmarshalStrict([]byte(c), &config); err == nil {
			if config.Kind == MicrovmMachineConfigKind {
				configs[config.Name] = &config
				continue
			}
		}
		_ = yaml.Unmarshal([]byte(c), &config) // this is to check if there is a bad spec in the file
		if config.Kind == MicrovmMachineConfigKind {
			return nil, fmt.Errorf("unable to unmarshall content from file due to: %v", err)
		}
	}
	if len(configs) == 0 {
		return nil, fmt.Errorf("unable to find kind %v in file", MicrovmMachineConfigKind)
	}
	return configs, nil
}
