package microvm

import (
	"fmt"
	"time"

	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/pkg/clusterapi"
	"github.com/aws/eks-anywhere/pkg/constants"
	"github.com/aws/eks-anywhere/pkg/crypto"
	"github.com/aws/eks-anywhere/pkg/providers"
	"github.com/aws/eks-anywhere/pkg/providers/common"
	"github.com/aws/eks-anywhere/pkg/templater"
	"github.com/aws/eks-anywhere/pkg/types"
)

func NewMicrovmTemplateBuilder(now types.NowFunc) providers.TemplateBuilder {
	return &MicrovmTemplateBuilder{
		now: now,
	}
}

type MicrovmTemplateBuilder struct {
	now types.NowFunc
}

func (d *MicrovmTemplateBuilder) WorkerMachineTemplateName(clusterName string) string {
	t := d.now().UnixNano() / int64(time.Millisecond)
	return fmt.Sprintf("%s-worker-node-template-%d", clusterName, t)
}

func (d *MicrovmTemplateBuilder) CPMachineTemplateName(clusterName string) string {
	t := d.now().UnixNano() / int64(time.Millisecond)
	return fmt.Sprintf("%s-control-plane-template-%d", clusterName, t)
}

func (d *MicrovmTemplateBuilder) EtcdMachineTemplateName(clusterName string) string {
	t := d.now().UnixNano() / int64(time.Millisecond)
	return fmt.Sprintf("%s-etcd-template-%d", clusterName, t)
}

func (d *MicrovmTemplateBuilder) GenerateCAPISpecControlPlane(clusterSpec *cluster.Spec, buildOptions ...providers.BuildMapOption) (content []byte, err error) {
	values := buildTemplateMapCP(clusterSpec)
	for _, buildOption := range buildOptions {
		buildOption(values)
	}

	bytes, err := templater.Execute(defaultCAPIConfigCP, values)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func (d *MicrovmTemplateBuilder) GenerateCAPISpecWorkers(clusterSpec *cluster.Spec, buildOptions ...providers.BuildMapOption) (content []byte, err error) {
	values := buildTemplateMapMD(clusterSpec)
	for _, buildOption := range buildOptions {
		buildOption(values)
	}

	bytes, err := templater.Execute(defaultClusterConfigMD, values)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func buildTemplateMapCP(clusterSpec *cluster.Spec) map[string]interface{} {
	bundle := clusterSpec.VersionsBundle
	etcdExtraArgs := clusterapi.SecureEtcdTlsCipherSuitesExtraArgs()
	sharedExtraArgs := clusterapi.SecureTlsCipherSuitesExtraArgs()
	apiServerExtraArgs := clusterapi.OIDCToExtraArgs(clusterSpec.OIDCConfig).
		Append(clusterapi.AwsIamAuthExtraArgs(clusterSpec.AWSIamConfig)).
		Append(clusterapi.PodIAMAuthExtraArgs(clusterSpec.Spec.PodIAMConfig)).
		Append(sharedExtraArgs)

	values := map[string]interface{}{
		"clusterName":                clusterSpec.Name,
		"controlPlaneEndpointIp":     clusterSpec.Spec.ControlPlaneConfiguration.Endpoint.Host,
		"controlPlaneReplicas":       clusterSpec.Spec.ControlPlaneConfiguration.Count,
		"kubernetesRepository":       bundle.KubeDistro.Kubernetes.Repository,
		"kubernetesVersion":          bundle.KubeDistro.Kubernetes.Tag,
		"etcdRepository":             bundle.KubeDistro.Etcd.Repository,
		"etcdVersion":                bundle.KubeDistro.Etcd.Tag,
		"corednsRepository":          bundle.KubeDistro.CoreDNS.Repository,
		"corednsVersion":             bundle.KubeDistro.CoreDNS.Tag,
		"kindNodeImage":              bundle.EksD.KindNode.VersionedImage(),
		"etcdExtraArgs":              etcdExtraArgs.ToPartialYaml(),
		"etcdCipherSuites":           crypto.SecureCipherSuitesString(),
		"apiserverExtraArgs":         apiServerExtraArgs.ToPartialYaml(),
		"controllermanagerExtraArgs": sharedExtraArgs.ToPartialYaml(),
		"schedulerExtraArgs":         sharedExtraArgs.ToPartialYaml(),
		"kubeletExtraArgs":           sharedExtraArgs.ToPartialYaml(),
		"externalEtcdVersion":        bundle.KubeDistro.EtcdVersion,
		"eksaSystemNamespace":        constants.EksaSystemNamespace,
		"auditPolicy":                common.GetAuditPolicy(),
		"podCidrs":                   clusterSpec.Spec.ClusterNetwork.Pods.CidrBlocks,
		"serviceCidrs":               clusterSpec.Spec.ClusterNetwork.Services.CidrBlocks,
		"kubeVipImage":               "ghcr.io/kube-vip/kube-vip:latest", // TODO: get this value from the bundle once we add it
	}

	if clusterSpec.Spec.ExternalEtcdConfiguration != nil {
		values["externalEtcd"] = true
		values["externalEtcdReplicas"] = clusterSpec.Spec.ExternalEtcdConfiguration.Count
	}
	if clusterSpec.AWSIamConfig != nil {
		values["awsIamAuth"] = true
	}

	if len(clusterSpec.Spec.ControlPlaneConfiguration.Taints) > 0 {
		values["controlPlaneTaints"] = clusterSpec.Spec.ControlPlaneConfiguration.Taints
	}

	return values
}

func buildTemplateMapMD(clusterSpec *cluster.Spec) map[string]interface{} {
	bundle := clusterSpec.VersionsBundle
	kubeletExtraArgs := clusterapi.SecureTlsCipherSuitesExtraArgs()

	values := map[string]interface{}{
		"clusterName":         clusterSpec.Name,
		"worker_replicas":     clusterSpec.Spec.WorkerNodeGroupConfigurations[0].Count,
		"kubernetesVersion":   bundle.KubeDistro.Kubernetes.Tag,
		"kindNodeImage":       bundle.EksD.KindNode.VersionedImage(),
		"eksaSystemNamespace": constants.EksaSystemNamespace,
		"kubeletExtraArgs":    kubeletExtraArgs.ToPartialYaml(),
	}
	return values
}
