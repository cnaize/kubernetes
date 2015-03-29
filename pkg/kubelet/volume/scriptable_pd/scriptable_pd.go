/*
Copyright 2014 Google Inc. All rights reserved.
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

package scriptable_pd

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"

	"github.com/cnaize/kubernetes/pkg/api"
	"github.com/cnaize/kubernetes/pkg/kubelet/volume"
	"github.com/cnaize/kubernetes/pkg/types"
	"github.com/golang/glog"
)

// This is the primary entrypoint for volume plugins.
func ProbeVolumePlugins() []volume.Plugin {
	return []volume.Plugin{&scriptablePersistentDiskPlugin{nil}}
}

type scriptablePersistentDiskPlugin struct {
	host volume.Host
}

var _ volume.Plugin = &scriptablePersistentDiskPlugin{}

const (
	scriptablePersistentDiskPluginName = "kubernetes.io/scriptable-pd"
)

func (plugin *scriptablePersistentDiskPlugin) Init(host volume.Host) {
	plugin.host = host
}

func (plugin *scriptablePersistentDiskPlugin) Name() string {
	return scriptablePersistentDiskPluginName
}

func (plugin *scriptablePersistentDiskPlugin) CanSupport(spec *api.Volume) bool {
	if spec.Source.ScriptablePersistentDisk != nil {
		return true
	}

	return false
}

func (plugin *scriptablePersistentDiskPlugin) NewBuilder(spec *api.Volume, podUID types.UID) (volume.Builder, error) {
	return &scriptablePersistentDisk{
		script:  spec.Source.ScriptablePersistentDisk.Script,
		params:  spec.Source.ScriptablePersistentDisk.Params,
		podUID:  podUID,
		volName: spec.Name,
		plugin:  plugin,
	}, nil
}

func (plugin *scriptablePersistentDiskPlugin) NewCleaner(volName string, podUID types.UID) (volume.Cleaner, error) {
	return &scriptablePersistentDisk{
		podUID:  podUID,
		volName: volName,
		plugin:  plugin,
	}, nil
}

// gcePersistentDisk volumes are disk resources provided by Google Compute Engine
// that are attached to the kubelet's host machine and exposed to the pod.
type scriptablePersistentDisk struct {
	script  string
	params  string
	volName string
	podUID  types.UID
	plugin  *scriptablePersistentDiskPlugin
}

// SetUp attaches the disk and bind mounts to the volume path.
func (pd *scriptablePersistentDisk) SetUp() error {
	return pd.SetUpAt(pd.GetPath())
}

// SetUpAt attaches the disk and bind mounts to the volume path.
func (pd *scriptablePersistentDisk) SetUpAt(dir string) error {
	glog.Infoln("SCIPTABLE DISK SETUPING")

	scriptParams, err := base64.StdEncoding.DecodeString(pd.params)
	if err != nil {
		return err
	}

	params := []string{"-c", pd.script}
	params = append(params, strings.Split(string(scriptParams), ";")...)

	if out, err := exec.Command("sh", params...).Output(); err != nil {
		return fmt.Errorf("can't execute script: %v\n", err)
	} else {
		glog.Infof("sctipt finished: %v\n", string(out))
	}

	return nil
}

func (pd *scriptablePersistentDisk) GetPath() string {
	name := scriptablePersistentDiskPluginName

	return pd.plugin.host.GetPodVolumeDir(pd.podUID, volume.EscapePluginName(name), pd.volName)
}

// Unmounts the bind mount, and detaches the disk only if the PD
// resource was the last reference to that disk on the kubelet.
func (pd *scriptablePersistentDisk) TearDown() error {
	return pd.TearDownAt(pd.GetPath())
}

// Unmounts the bind mount, and detaches the disk only if the PD
// resource was the last reference to that disk on the kubelet.
func (pd *scriptablePersistentDisk) TearDownAt(dir string) error {
	return nil
}
