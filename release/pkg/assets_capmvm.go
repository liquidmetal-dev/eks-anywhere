package pkg

import (
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
)

const capmvmProjectPath = "projects/weaveworks/cluster-api-provider-microvm"

// GetCapmvmAssets returns the eks-a artifacts for CAPMVM
func (r *ReleaseConfig) GetCapmvmAssets() ([]Artifact, error) {
	gitTag, err := r.readGitTag(capmvmProjectPath, r.BuildRepoBranchName)
	if err != nil {
		return nil, errors.Cause(err)
	}

	name := "cluster-api-provider-microvm"
	repoName := "weaveworks/cluster-api-provider-microvm"
	tagOptions := map[string]string{
		"gitTag":      gitTag,
		"projectPath": capmvmProjectPath,
	}

	sourceImageUri, sourcedFromBranch, err := r.GetSourceImageURI(name, repoName, tagOptions)
	if err != nil {
		return nil, errors.Cause(err)
	}
	if sourcedFromBranch != r.BuildRepoBranchName {
		gitTag, err = r.readGitTag(capmvmProjectPath, sourcedFromBranch)
		if err != nil {
			return nil, errors.Cause(err)
		}
		tagOptions["gitTag"] = gitTag
	}
	releaseImageUri, err := r.GetReleaseImageURI(name, repoName, tagOptions)
	if err != nil {
		return nil, errors.Cause(err)
	}

	imageArtifact := &ImageArtifact{
		AssetName:         name,
		SourceImageURI:    sourceImageUri,
		ReleaseImageURI:   releaseImageUri,
		Arch:              []string{"amd64"},
		OS:                "linux",
		GitTag:            gitTag,
		ProjectPath:       capmvmProjectPath,
		SourcedFromBranch: sourcedFromBranch,
	}
	artifacts := []Artifact{Artifact{Image: imageArtifact}}

	imageTagOverrides := []ImageTagOverride{
		{
			Repository: repoName,
			ReleaseUri: imageArtifact.ReleaseImageURI,
		},
	}

	manifestList := []string{
		"infrastructure-components.yaml",
		"cluster-template.yaml",
		"metadata.yaml",
	}

	for _, manifest := range manifestList {
		var sourceS3Prefix string
		var releaseS3Path string
		latestPath := getLatestUploadDestination(sourcedFromBranch)

		if r.DevRelease || r.ReleaseEnvironment == "development" {
			sourceS3Prefix = fmt.Sprintf("%s/%s/manifests/infrastructure-microvm/%s", capmvmProjectPath, latestPath, gitTag)
		} else {
			sourceS3Prefix = fmt.Sprintf("releases/bundles/%d/artifacts/cluster-api-provider-microvm/manifests/infrastructure-microvm/%s", r.BundleNumber, gitTag)
		}

		if r.DevRelease {
			releaseS3Path = fmt.Sprintf("artifacts/%s/cluster-api-provider-microvm/manifests/infrastructure-microvm/%s", r.DevReleaseUriVersion, gitTag)
		} else {
			releaseS3Path = fmt.Sprintf("releases/bundles/%d/artifacts/cluster-api-provider-microvm/manifests/infrastructure-microvm/%s", r.BundleNumber, gitTag)
		}

		cdnURI, err := r.GetURI(filepath.Join(
			releaseS3Path,
			manifest))
		if err != nil {
			return nil, errors.Cause(err)
		}

		manifestArtifact := &ManifestArtifact{
			SourceS3Key:       manifest,
			SourceS3Prefix:    sourceS3Prefix,
			ArtifactPath:      filepath.Join(r.ArtifactDir, "capmvm-manifests", r.BuildRepoHead),
			ReleaseName:       manifest,
			ReleaseS3Path:     releaseS3Path,
			ReleaseCdnURI:     cdnURI,
			ImageTagOverrides: imageTagOverrides,
			GitTag:            gitTag,
			ProjectPath:       capmvmProjectPath,
		}
		artifacts = append(artifacts, Artifact{Manifest: manifestArtifact})
	}

	return artifacts, nil
}
