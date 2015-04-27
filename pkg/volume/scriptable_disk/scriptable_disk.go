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

package scriptable_disk

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/types"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/volume"
	"github.com/golang/glog"
)

// This is the primary entrypoint for volume plugins.
func ProbeVolumePlugins() []volume.VolumePlugin {
	return []volume.VolumePlugin{&scriptableDiskPlugin{nil}}
}

type scriptableDiskPlugin struct {
	host volume.VolumeHost
}

var _ volume.VolumePlugin = &scriptableDiskPlugin{}

const (
	scriptableDiskPluginName = "kubernetes.io/scriptable-disk"
	scriptsDir               = "/var/lib/kuberdock/scripts"
)

func (plugin *scriptableDiskPlugin) Init(host volume.VolumeHost) {
	plugin.host = host
}

func (plugin *scriptableDiskPlugin) Name() string {
	return scriptableDiskPluginName
}

func (plugin *scriptableDiskPlugin) CanSupport(spec *api.Volume) bool {
	if spec.VolumeSource.ScriptableDisk != nil {
		return true
	}

	return false
}

func (plugin *scriptableDiskPlugin) NewBuilder(spec *api.Volume, podRef *api.ObjectReference, _ volume.VolumeOptions) (volume.Builder, error) {
	return &scriptableDisk{
		pathToScript: spec.VolumeSource.ScriptableDisk.PathToScript,
		params:       spec.VolumeSource.ScriptableDisk.Params,
		podRef:       podRef,
		volName:      spec.Name,
		plugin:       plugin,
	}, nil
}

func (plugin *scriptableDiskPlugin) NewCleaner(name string, podUID types.UID) (volume.Cleaner, error) {
	return &scriptableDisk{
		podRef:  &api.ObjectReference{UID: podUID},
		volName: name,
		plugin:  plugin,
	}, nil
}

type scriptableDisk struct {
	pathToScript string
	params       string
	volName      string
	podRef       *api.ObjectReference
	plugin       *scriptableDiskPlugin
}

func (sd *scriptableDisk) SetUp() error {
	return sd.SetUpAt(sd.GetPath())
}

func (sd *scriptableDisk) SetUpAt(dir string) error {
	scriptParams, err := base64.StdEncoding.DecodeString(sd.params)
	if err != nil {
		return err
	}

	params := []string{filepath.Join(scriptsDir, sd.pathToScript)}
	params = append(params, strings.Split(string(scriptParams), ";")...)

	if out, err := exec.Command("sh", params...).Output(); err != nil {
		return fmt.Errorf("can't execute script: %v\n", err)
	} else {
		glog.Infof("script finished: %v\n", string(out))
	}

	return nil
}

func (sd *scriptableDisk) GetPath() string {
	name := scriptableDiskPluginName
	return sd.plugin.host.GetPodVolumeDir(sd.podRef.UID, util.EscapeQualifiedNameForDisk(name), sd.volName)
}

func (sd *scriptableDisk) TearDown() error {
	return sd.TearDownAt(sd.GetPath())
}

func (sd *scriptableDisk) TearDownAt(dir string) error {
	return nil
}
