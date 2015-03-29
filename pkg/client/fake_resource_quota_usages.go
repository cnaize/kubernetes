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

package client

import (
	"github.com/cnaize/kubernetes/pkg/api"
)

// FakeResourceQuotaUsages implements ResourceQuotaUsageInterface. Meant to be embedded into a struct to get a default
// implementation. This makes faking out just the methods you want to test easier.
type FakeResourceQuotaUsages struct {
	Fake      *Fake
	Namespace string
}

func (c *FakeResourceQuotaUsages) Create(resourceQuotaUsage *api.ResourceQuotaUsage) error {
	c.Fake.Actions = append(c.Fake.Actions, FakeAction{Action: "create-resourceQuotaUsage"})
	return nil
}
