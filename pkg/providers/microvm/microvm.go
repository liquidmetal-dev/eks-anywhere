package microvm

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"time"

	"github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/bootstrapper"
	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/constants"
	"github.com/aws/eks-anywhere/pkg/logger"
	"github.com/aws/eks-anywhere/pkg/providers"
	"github.com/aws/eks-anywhere/pkg/retrier"
	"github.com/aws/eks-anywhere/pkg/types"
	releasev1alpha1 "github.com/aws/eks-anywhere/release/api/v1alpha1"
)

const (
	hostEndPointKey = "HOST_ENDPOINT"
	//controlPlaneVIPKey = "CONTROL_PLANE_VIP"

	maxRetries    = 30
	backOffPeriod = 5 * time.Second
)

//go:embed config/template-cp.yaml
var defaultCAPIConfigCP string

//go:embed config/template-md.yaml
var defaultClusterConfigMD string

//go:embed config/coredns.yaml
var coreDNSConfig string

var (
	eksaMicrovmResourceType        = fmt.Sprintf("microvmdatacenterconfigs.%s", v1alpha1.GroupVersion.Group)
	eksaMicrovmMachineResourceType = fmt.Sprintf("microvmmachineconfigs.%s", v1alpha1.GroupVersion.Group)
	requiredEnvs                   = []string{hostEndPointKey}
)

type provider struct {
	clusterConfig          *v1alpha1.Cluster
	datacenterConfig       *v1alpha1.MicrovmDatacenterConfig
	machineConfigs         map[string]*v1alpha1.MicrovmMachineConfig
	controlPlaneSshAuthKey string
	workerSshAuthKey       string
	providerKubectlClient  ProviderKubectlClient
	templateBuilder        *MicrovmTemplateBuilder
	Retrier                *retrier.Retrier
}

type ProviderKubectlClient interface {
	ApplyKubeSpecFromBytes(ctx context.Context, cluster *types.Cluster, data []byte) error
	WaitForDeployment(ctx context.Context, cluster *types.Cluster, timeout string, condition string, target string, namespace string) error
}

func NewProvider(datacenterConfig *v1alpha1.MicrovmDatacenterConfig, machineConfigs map[string]*v1alpha1.MicrovmMachineConfig, clusterConfig *v1alpha1.Cluster, providerKubectlClient ProviderKubectlClient, now types.NowFunc) *provider {
	var controlPlaneMachineSpec, workerNodeGroupMachineSpec *v1alpha1.MicrovmMachineConfigSpec
	if clusterConfig.Spec.ControlPlaneConfiguration.MachineGroupRef != nil && machineConfigs[clusterConfig.Spec.ControlPlaneConfiguration.MachineGroupRef.Name] != nil {
		controlPlaneMachineSpec = &machineConfigs[clusterConfig.Spec.ControlPlaneConfiguration.MachineGroupRef.Name].Spec
	}
	if len(clusterConfig.Spec.WorkerNodeGroupConfigurations) > 0 && clusterConfig.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef != nil && machineConfigs[clusterConfig.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name] != nil {
		workerNodeGroupMachineSpec = &machineConfigs[clusterConfig.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name].Spec
	}

	return &provider{
		clusterConfig:         clusterConfig,
		datacenterConfig:      datacenterConfig,
		machineConfigs:        machineConfigs,
		providerKubectlClient: providerKubectlClient,
		templateBuilder: &MicrovmTemplateBuilder{
			now:                        now,
			datacenterSpec:             &datacenterConfig.Spec,
			controlPlaneMachineSpec:    controlPlaneMachineSpec,
			workerNodeGroupMachineSpec: workerNodeGroupMachineSpec,
		},
		Retrier: retrier.NewWithMaxRetries(maxRetries, backOffPeriod),
	}
}

func (p *provider) BootstrapClusterOpts() ([]bootstrapper.BootstrapClusterOption, error) {
	env := map[string]string{}
	if p.clusterConfig.Spec.ProxyConfiguration != nil {
		noProxy := fmt.Sprintf("%s,%s", p.clusterConfig.Spec.ControlPlaneConfiguration.Endpoint.Host, p.datacenterConfig.Spec.FlintlockURL)
		for _, s := range p.clusterConfig.Spec.ProxyConfiguration.NoProxy {
			if s != "" {
				noProxy += "," + s
			}
		}
		env["HTTP_PROXY"] = p.clusterConfig.Spec.ProxyConfiguration.HttpProxy
		env["HTTPS_PROXY"] = p.clusterConfig.Spec.ProxyConfiguration.HttpsProxy
		env["NO_PROXY"] = noProxy
	}
	// TODO: do we need: bootstrapper.WithExtraDockerMounts()
	return []bootstrapper.BootstrapClusterOption{bootstrapper.WithEnv(env)}, nil
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
	if err := setEnvVars(p.datacenterConfig); err != nil {
		return fmt.Errorf("failed setup and validations: %v", err)
	}
	p.controlPlaneSshAuthKey = p.machineConfigs[p.clusterConfig.Spec.ControlPlaneConfiguration.MachineGroupRef.Name].Spec.Users[0].SshAuthorizedKeys[0]
	p.workerSshAuthKey = p.machineConfigs[p.clusterConfig.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name].Spec.Users[0].SshAuthorizedKeys[0]
	return nil
}

func (p *provider) SetupAndValidateDeleteCluster(ctx context.Context) error {
	// TODO: validations?
	if err := setEnvVars(p.datacenterConfig); err != nil {
		return fmt.Errorf("failed setup and validations: %v", err)
	}
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
	// TODO: Figure out if something is needed here
	var configs []providers.MachineConfig
	controlPlaneMachineName := p.clusterConfig.Spec.ControlPlaneConfiguration.MachineGroupRef.Name
	workerMachineName := p.clusterConfig.Spec.WorkerNodeGroupConfigurations[0].MachineGroupRef.Name
	p.machineConfigs[controlPlaneMachineName].Annotations = map[string]string{p.clusterConfig.ControlPlaneAnnotation(): "true"}
	if p.clusterConfig.IsManaged() {
		p.machineConfigs[controlPlaneMachineName].SetManagement(p.clusterConfig.ManagedBy())
	}

	configs = append(configs, p.machineConfigs[controlPlaneMachineName])
	if workerMachineName != controlPlaneMachineName {
		configs = append(configs, p.machineConfigs[workerMachineName])
		if p.clusterConfig.IsManaged() {
			p.machineConfigs[workerMachineName].SetManagement(p.clusterConfig.ManagedBy())
		}
	}
	return configs
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
	// logger.Info("Waiting got CNI (cilium) to be ready")
	// err := p.Retrier.Retry(
	// 	func() error {
	// 		return p.providerKubectlClient.WaitForDeployment(ctx, cluster, "5m", "Available", "cilium-operator", "kube-system")
	// 	},
	// )
	// if err != nil {
	// 	return fmt.Errorf("applying coredns configmap: %w", err)
	// }

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

func setEnvVars(datacenterConfig *v1alpha1.MicrovmDatacenterConfig) error {
	if err := os.Setenv(hostEndPointKey, datacenterConfig.Spec.FlintlockURL); err != nil {
		return fmt.Errorf("unable to set %s: %v", hostEndPointKey, err)
	}

	return nil
}
