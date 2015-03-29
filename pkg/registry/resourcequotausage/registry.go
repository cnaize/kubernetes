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

package resourcequotausage

import (
	"github.com/cnaize/kubernetes/pkg/api"
)

// Registry contains the functions needed to support a ResourceQuotaUsage
type Registry interface {
	// ApplyStatus should update the ResourceQuota.Status with latest observed state.
	// This should be atomic, and idempotent based on the ResourceVersion
	ApplyStatus(ctx api.Context, usage *api.ResourceQuotaUsage) error
}
