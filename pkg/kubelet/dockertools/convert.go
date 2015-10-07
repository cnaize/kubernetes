/*
Copyright 2015 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dockertools

import (
	"fmt"
	"strings"

	docker "github.com/cnaize/go-dockerclient"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	kubeclient "k8s.io/kubernetes/pkg/client"
	"k8s.io/kubernetes/pkg/fields"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/types"
)

// This file contains helper functions to convert docker API types to runtime
// (kubecontainer) types.

// Converts docker.APIContainers to kubecontainer.Container.
func toRuntimeContainer(client DockerInterface, kc kubeclient.Interface, c *docker.APIContainers) (*kubecontainer.Container, error) {
	if c == nil {
		return nil, fmt.Errorf("unable to convert a nil pointer to a runtime container")
	}

	inspectResult, err := client.InspectContainer(c.ID)
	if err != nil {
		return nil, err
	}

	dockerName, hash, err := getDockerContainerNameInfo(c)
	if err != nil {
		return nil, err
	}

	hostConfig := inspectResult.HostConfig
	limits := make(api.ResourceList)
	limits[api.ResourceMemory] = *resource.NewQuantity(hostConfig.Memory, resource.DecimalSI)
	limits[api.ResourceCPU] = *resource.NewMilliQuantity(SharesToMulliCPU(hostConfig.CPUShares), resource.DecimalSI)

	if dockerName.ContainerName != "POD" {
		nameNamespace := strings.Split(dockerName.PodFullName, "_")

		pods, err := kc.Pods(nameNamespace[1]).
			List(labels.Everything(), fields.OneTermEqualSelector(kubeclient.ObjectNameField, nameNamespace[0]))
		if err != nil {
			return nil, err
		}

		if len(pods.Items) == 1 {
			var container *api.Container
			for _, cr := range pods.Items[0].Spec.Containers {
				if cr.Name == dockerName.ContainerName {
					container = &cr
					break
				}
			}

			if container != nil {
				hash = kubecontainer.HashContainer(container)
			}
		}
	}

	return &kubecontainer.Container{
		Limits:  limits,
		ID:      types.UID(c.ID),
		Name:    dockerName.ContainerName,
		Image:   c.Image,
		Hash:    hash,
		Created: c.Created,
	}, nil
}

// Converts docker.APIImages to kubecontainer.Image.
func toRuntimeImage(image *docker.APIImages) (*kubecontainer.Image, error) {
	if image == nil {
		return nil, fmt.Errorf("unable to convert a nil pointer to a runtime image")
	}

	return &kubecontainer.Image{
		ID:   image.ID,
		Tags: image.RepoTags,
		Size: image.VirtualSize,
	}, nil
}
