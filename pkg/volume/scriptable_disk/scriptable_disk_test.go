/*
Copyright 2014 The Kubernetes Authors All rights reserved.

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
	"os"
	"path"
	"testing"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/types"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/volume"
)

// The dir where volumes will be stored.
const (
	basePath       = "/tmp/fake"
	plugName       = "kubernetes.io/scriptable-disk"
	testScript     = "echo $0"
	testScriptName = "k8s_auto_test.sh"
)

// Construct an instance of a plugin, by name.
func makePluginUnderTest(t *testing.T) volume.VolumePlugin {
	plugMgr := volume.VolumePluginMgr{}
	plugMgr.InitPlugins(ProbeVolumePlugins(), volume.NewFakeVolumeHost(basePath, nil, nil))

	plug, err := plugMgr.FindPluginByName(plugName)
	if err != nil {
		t.Errorf("Can't find the plugin by name")
	}
	return plug
}

func TestCanSupport(t *testing.T) {
	plug := makePluginUnderTest(t)

	if plug.Name() != plugName {
		t.Errorf("Wrong name: %s", plug.Name())
	}
	if !plug.CanSupport(&volume.Spec{Name: "foo", VolumeSource: api.VolumeSource{ScriptableDisk: &api.ScriptableDiskVolumeSource{}}}) {
		t.Errorf("Expected true")
	}
	if plug.CanSupport(&volume.Spec{Name: "foo", VolumeSource: api.VolumeSource{}}) {
		t.Errorf("Expected false")
	}
}

func TestPlugin(t *testing.T) {
	plug := makePluginUnderTest(t)

	if _, err := os.Stat(scriptsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(scriptsDir, 0744); err != nil {
			t.Errorf("Failed to create scripts folder: %v", err)
		}
	}
	scriptFile, err := os.Create(scriptsDir + "/" + testScriptName)
	if err != nil {
		t.Errorf("Failed to create test script file: %v", err)
	}
	defer scriptFile.Close()
	if _, err := scriptFile.WriteString(testScript); err != nil {
		t.Errorf("Failed write script into file: %v", err)
	}

	spec := &api.Volume{
		Name:         "vol1",
		VolumeSource: api.VolumeSource{ScriptableDisk: &api.ScriptableDiskVolumeSource{PathToScript: testScriptName}},
	}
	pod := &api.Pod{ObjectMeta: api.ObjectMeta{UID: types.UID("poduid")}}
	builder, err := plug.(*scriptableDiskPlugin).NewBuilder(volume.NewSpecFromVolume(spec), pod, volume.VolumeOptions{""}, nil)
	if err != nil {
		t.Errorf("Failed to make a new Builder: %v", err)
	}
	if builder == nil {
		t.Errorf("Got a nil Builder")
	}

	volPath := builder.GetPath()
	if volPath != path.Join(basePath, "pods/poduid/volumes/kubernetes.io~scriptable-disk/vol1") {
		t.Errorf("Got unexpected path: %s", volPath)
	}

	if err := builder.SetUp(); err != nil {
		t.Errorf("Expected success, got: %v", err)
	}

	cleaner, err := plug.(*scriptableDiskPlugin).NewCleaner("vol1", types.UID("poduid"), nil)
	if err != nil {
		t.Errorf("Failed to make a new Cleaner: %v", err)
	}
	if cleaner == nil {
		t.Errorf("Got a nil Cleaner")
	}

	if err := cleaner.TearDown(); err != nil {
		t.Errorf("Expected success, got: %v", err)
	}
	if _, err := os.Stat(scriptsTmpDir + "/" + "poduid"); os.IsExist(err) {
		t.Errorf("TearDown() failed, script file still exists: %s", volPath)
	}

	if err := os.Remove(scriptsDir + "/" + testScriptName); err != nil {
		t.Errorf("Failed to remove script file: %v", testScriptName)
	}
}
