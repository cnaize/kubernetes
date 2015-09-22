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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/types"
	"k8s.io/kubernetes/pkg/util"
	"k8s.io/kubernetes/pkg/util/mount"
	"k8s.io/kubernetes/pkg/volume"
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
)

var (
	scriptsDir    = filepath.Join("/", "var", "lib", "kuberdock", "scripts")
	scriptsTmpDir = filepath.Join(scriptsDir, "tmp")
)

func (plugin *scriptableDiskPlugin) Init(host volume.VolumeHost) {
	plugin.host = host
}

func (plugin *scriptableDiskPlugin) Name() string {
	return scriptableDiskPluginName
}

func (plugin *scriptableDiskPlugin) CanSupport(spec *volume.Spec) bool {
	if spec.VolumeSource.ScriptableDisk != nil {
		glog.Infoln("using scriptable disk")
		return true
	}

	return false
}

func (plugin *scriptableDiskPlugin) NewBuilder(spec *volume.Spec, pod *api.Pod, _ volume.VolumeOptions, _ mount.Interface) (volume.Builder, error) {
	pathToScript := spec.VolumeSource.ScriptableDisk.PathToScript
	podUID := pod.UID

	if _, err := os.Stat(scriptsTmpDir); os.IsNotExist(err) {
		os.MkdirAll(scriptsTmpDir, 0744)
	}

	script, err := ioutil.ReadFile(filepath.Join(scriptsDir, pathToScript))
	if err != nil {
		return nil, err
	}

	if err := ioutil.WriteFile(filepath.Join(scriptsTmpDir, string(podUID)), script, 0744); err != nil {
		return nil, err
	}

	return &scriptableDisk{
		pathToScript: pathToScript,
		params:       spec.VolumeSource.ScriptableDisk.Params,
		podUID:       podUID,
		volName:      spec.Name,
		plugin:       plugin,
	}, nil
}

func (plugin *scriptableDiskPlugin) NewCleaner(name string, podUID types.UID, _ mount.Interface) (volume.Cleaner, error) {
	return &scriptableDisk{
		podUID:  podUID,
		volName: name,
		plugin:  plugin,
	}, nil
}

type scriptableDisk struct {
	pathToScript string
	params       string
	volName      string
	podUID       types.UID
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

	params := []string{filepath.Join(scriptsDir, sd.pathToScript), string(sd.podUID)}
	params = append(params, strings.Split(string(scriptParams), ";")...)

	if out, err := exec.Command("sh", params...).Output(); err != nil {
		return fmt.Errorf("can't execute setup script: %v\n", err)
	} else {
		glog.Infof("setup script finished: %v\n", string(out))
	}

	return nil
}

func (sd *scriptableDisk) GetPath() string {
	name := scriptableDiskPluginName
	return sd.plugin.host.GetPodVolumeDir(sd.podUID, util.EscapeQualifiedNameForDisk(name), sd.volName)
}

func (sd *scriptableDisk) TearDown() error {
	return sd.TearDownAt(sd.GetPath())
}

func (sd *scriptableDisk) TearDownAt(dir string) error {
	podUID := string(sd.podUID)
	pathToScript := filepath.Join(scriptsTmpDir, podUID)
	params := []string{pathToScript, "umount", podUID}

	if out, err := exec.Command("sh", params...).Output(); err != nil {
		return fmt.Errorf("can't execute tear down script: %v\n", err)
	} else {
		glog.Infof("tear down script finished: %v\n", string(out))
	}

	if err := os.Remove(pathToScript); err != nil {
		glog.Warningf("can't remove tmp script: %v, for podUID: %v\n", err, podUID)
	}

	return nil
}
