package microvm

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/bootstrapper"
	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/constants"
	"github.com/aws/eks-anywhere/pkg/logger"
	"github.com/aws/eks-anywhere/pkg/providers"
	"github.com/aws/eks-anywhere/pkg/types"
	releasev1alpha1 "github.com/aws/eks-anywhere/release/api/v1alpha1"
)

const (
	flintlockIPKey = "FLINTLOCK_IP"
)

//go:embed config/template-cp.yaml
var defaultCAPIConfigCP string

//go:embed config/template-md.yaml
var defaultClusterConfigMD string

var (
	eksaMicrovmResourceType        = fmt.Sprintf("microvmdatacenterconfigs.%s", v1alpha1.GroupVersion.Group)
	eksaMicrovmMachineResourceType = fmt.Sprintf("microvmmachineconfigs.%s", v1alpha1.GroupVersion.Group)
	requiredEnvs                   = []string{flintlockIPKey}
)

type provider struct {
	clusterConfig         *v1alpha1.Cluster
	datacenterConfig      *v1alpha1.MicrovmDatacenterConfig
	machineConfigs        map[string]*v1alpha1.MicrovmMachineConfig
	sshKey                string
	providerKubectlClient ProviderKubectlClient
	templateBuilder       *MicrovmTemplateBuilder
}

type ProviderKubectlClient interface {
	ApplyHardware(ctx context.Context, hardwareYaml string, kubeConfFile string) error
}

func NewProvider(datacenterConfig *v1alpha1.MicrovmDatacenterConfig, machineConfigs map[string]*v1alpha1.MicrovmMachineConfig, clusterConfig *v1alpha1.Cluster, providerKubectlClient ProviderKubectlClient, now types.NowFunc) *provider {
	return &provider{
		clusterConfig:         clusterConfig,
		datacenterConfig:      datacenterConfig,
		machineConfigs:        machineConfigs,
		providerKubectlClient: providerKubectlClient,
		templateBuilder: &MicrovmTemplateBuilder{
			now: now,
		},
	}
}

func (p *provider) BootstrapClusterOpts() ([]bootstrapper.BootstrapClusterOption, error) {
	//TODO: do we need anything here?
	return nil, nil
}

func (p *provider) BootstrapSetup(ctx context.Context, clusterConfig *v1alpha1.Cluster, cluster *types.Cluster) error {
	return nil
}

func (p *provider) Name() string {
	return constants.MicrovmProviderName
}

func (p *provider) DatacenterResourceType() string {
	return eksaMicrovmResourceType
}

func (p *provider) MachineResourceType() string {
	return ""
}

func (p *provider) DeleteResources(_ context.Context, _ *cluster.Spec) error {
	return nil
}

func (p *provider) SetupAndValidateCreateCluster(ctx context.Context, clusterSpec *cluster.Spec) error {
	logger.Info("Warning: The microvm infrastructure provider is still in development and should only be used for testing/dev purposes")
	//if clusterSpec.Spec.ControlPlaneConfiguration.Endpoint != nil && clusterSpec.Spec.ControlPlaneConfiguration.Endpoint.Host != "" {
	//	return fmt.Errorf("specifying endpoint host configuration in Cluster is not supported")
	//}
	return nil
}

func (p *provider) SetupAndValidateDeleteCluster(ctx context.Context) error {
	return nil
}

func (p *provider) SetupAndValidateUpgradeCluster(ctx context.Context, _ *types.Cluster, _ *cluster.Spec) error {
	return nil
}

func (p *provider) UpdateSecrets(ctx context.Context, cluster *types.Cluster) error {
	// Not implemented
	return nil
}

func (p *provider) GenerateCAPISpecForCreate(ctx context.Context, cluster *types.Cluster, clusterSpec *cluster.Spec) (controlPlaneSpec, workersSpec []byte, err error) {
	controlPlaneSpec, workersSpec, err = p.generateCAPISpecForCreate(ctx, cluster, clusterSpec)
	if err != nil {
		return nil, nil, fmt.Errorf("error generating cluster api spec contents: %v", err)
	}
	return controlPlaneSpec, workersSpec, nil
}

func (p *provider) GenerateCAPISpecForUpgrade(ctx context.Context, bootstrapCluster, workloadCluster *types.Cluster, currentSpec, newClusterSpec *cluster.Spec) (controlPlaneSpec, workersSpec []byte, err error) {
	//TODO: implement
	return nil, nil, nil
}

func (p *provider) GenerateStorageClass() []byte {
	return nil
}

func (p *provider) GenerateMHC() ([]byte, error) {
	return []byte{}, nil
}

func (p *provider) UpdateKubeConfig(content *[]byte, clusterName string) error {
	return nil
}

func (p *provider) Version(clusterSpec *cluster.Spec) string {
	return clusterSpec.VersionsBundle.Docker.Version
}

func (p *provider) EnvMap() (map[string]string, error) {
	// TODO: determine if any env vars are needed and add them to requiredEnvs
	envMap := make(map[string]string)
	for _, key := range requiredEnvs {
		if env, ok := os.LookupEnv(key); ok && len(env) > 0 {
			envMap[key] = env
		} else {
			return envMap, fmt.Errorf("warning required env not set %s", key)
		}
	}
	return envMap, nil
}

func (p *provider) GetDeployments() map[string][]string {
	return map[string][]string{
		"capmvm-system": {"capmvm-controller-manager"},
	}
}

func (p *provider) GetInfrastructureBundle(clusterSpec *cluster.Spec) *types.InfrastructureBundle {
	bundle := clusterSpec.VersionsBundle
	folderName := fmt.Sprintf("infrastructure-microvm/%s/", bundle.Microvm.Version)

	infraBundle := types.InfrastructureBundle{
		FolderName: folderName,
		Manifests: []releasev1alpha1.Manifest{
			bundle.Microvm.Components,
			bundle.Microvm.Metadata,
			bundle.Microvm.ClusterTemplate,
		},
	}

	return &infraBundle
}

func (p *provider) DatacenterConfig() providers.DatacenterConfig {
	return p.datacenterConfig
}

func (p *provider) MachineConfigs() []providers.MachineConfig {
	return nil
}

func (p *provider) ValidateNewSpec(_ context.Context, _ *types.Cluster, _ *cluster.Spec) error {
	return nil
}

func (p *provider) ChangeDiff(currentSpec, newSpec *cluster.Spec) *types.ComponentChangeDiff {
	if currentSpec.VersionsBundle.Microvm.Version == newSpec.VersionsBundle.Microvm.Version {
		return nil
	}

	return &types.ComponentChangeDiff{
		ComponentName: constants.MicrovmProviderName,
		NewVersion:    newSpec.VersionsBundle.Microvm.Version,
		OldVersion:    currentSpec.VersionsBundle.Microvm.Version,
	}
}

func (p *provider) RunPostControlPlaneUpgrade(ctx context.Context, oldClusterSpec *cluster.Spec, clusterSpec *cluster.Spec, workloadCluster *types.Cluster, managementCluster *types.Cluster) error {
	return nil
}

func (p *provider) UpgradeNeeded(_ context.Context, _, _ *cluster.Spec) (bool, error) {
	return false, nil
}

func (p *provider) RunPostControlPlaneCreation(ctx context.Context, clusterSpec *cluster.Spec, cluster *types.Cluster) error {
	return nil
}

func NeedsNewControlPlaneTemplate(oldSpec, newSpec *cluster.Spec) bool {
	return (oldSpec.Cluster.Spec.KubernetesVersion != newSpec.Cluster.Spec.KubernetesVersion) || (oldSpec.Bundles.Spec.Number != newSpec.Bundles.Spec.Number)
}

func NeedsNewWorkloadTemplate(oldSpec, newSpec *cluster.Spec) bool {
	return (oldSpec.Cluster.Spec.KubernetesVersion != newSpec.Cluster.Spec.KubernetesVersion) || (oldSpec.Bundles.Spec.Number != newSpec.Bundles.Spec.Number)
}

func NeedsNewEtcdTemplate(oldSpec, newSpec *cluster.Spec) bool {
	return (oldSpec.Cluster.Spec.KubernetesVersion != newSpec.Cluster.Spec.KubernetesVersion) || (oldSpec.Bundles.Spec.Number != newSpec.Bundles.Spec.Number)
}

func (p *provider) generateCAPISpecForCreate(ctx context.Context, cluster *types.Cluster, clusterSpec *cluster.Spec) (controlPlaneSpec, workersSpec []byte, err error) {
	clusterName := clusterSpec.ObjectMeta.Name

	cpOpt := func(values map[string]interface{}) {
		values["controlPlaneTemplateName"] = p.templateBuilder.CPMachineTemplateName(clusterName)
		values["etcdTemplateName"] = p.templateBuilder.EtcdMachineTemplateName(clusterName)
	}
	controlPlaneSpec, err = p.templateBuilder.GenerateCAPISpecControlPlane(clusterSpec, cpOpt)
	if err != nil {
		return nil, nil, err
	}
	workersOpts := func(values map[string]interface{}) {
		values["workloadTemplateName"] = p.templateBuilder.WorkerMachineTemplateName(clusterName)
		//TODO: add ssh key?
	}
	workersSpec, err = p.templateBuilder.GenerateCAPISpecWorkers(clusterSpec, workersOpts)
	if err != nil {
		return nil, nil, err
	}
	return controlPlaneSpec, workersSpec, nil
}
