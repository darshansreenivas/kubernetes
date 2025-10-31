/*
Copyright 2025 The Kubernetes Authors.

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

/*
Copyright 2025 The Kubernetes Authors.

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

package csinode

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	apitesting "k8s.io/kubernetes/pkg/api/testing"
	"k8s.io/kubernetes/pkg/apis/storage"
)

func TestDeclarativeValidate(t *testing.T) {
	// CSINode had v1beta1 → v1, keep both to catch skew
	apiVersions := []string{"v1beta1", "v1"}
	for _, apiVersion := range apiVersions {
		t.Run(apiVersion, func(t *testing.T) {
			testDeclarativeValidate(t, apiVersion)
		})
	}
}

func testDeclarativeValidate(t *testing.T, apiVersion string) {
	ctx := genericapirequest.WithRequestInfo(
		genericapirequest.NewDefaultContext(),
		&genericapirequest.RequestInfo{
			APIPrefix:         "apis",
			APIGroup:          "storage.k8s.io",
			APIVersion:        apiVersion,
			Resource:          "csinodes",
			IsResourceRequest: true,
			Verb:              "create",
		},
	)

	testCases := map[string]struct {
		input        storage.CSINode
		expectedErrs field.ErrorList
	}{
		"valid": {
			input: mkValidCSINodeDriverNode(),
		},
		"missing nodeID": {
			input: mkValidCSINodeDriverNode(func(d *storage.CSINodeDriver) {
				d.NodeID = ""
			}),
			expectedErrs: field.ErrorList{
				field.Required(
					field.NewPath("spec").Child("drivers").Index(0).Child("nodeID"),
					"",
				),
			},
		},
		"missing name": {
			input: mkValidCSINodeDriverNode(func(driver *storage.CSINodeDriver) {
				driver.Name = ""
			}),
			expectedErrs: field.ErrorList{
				field.Required(
					field.NewPath("spec").Child("drivers").Index(0).Child("name"),
					"",
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			apitesting.VerifyValidationEquivalence(
				t,
				ctx,
				&tc.input,
				Strategy.Validate,
				tc.expectedErrs,
			)
		})
	}
}

func TestDeclarativeValidateUpdate(t *testing.T) {
	apiVersions := []string{"v1beta1", "v1"} // CSINode existed as v1beta1 → v1
	for _, apiVersion := range apiVersions {
		t.Run(apiVersion, func(t *testing.T) {
			testDeclarativeValidateUpdate(t, apiVersion)
		})
	}
}

func testDeclarativeValidateUpdate(t *testing.T, apiVersion string) {
	// common path bits
	driverPath := field.NewPath("spec").Child("drivers").Index(0)

	testCases := map[string]struct {
		oldObj       storage.CSINode
		updateObj    storage.CSINode
		expectedErrs field.ErrorList
	}{
		"invalid update (driver name changed)": {
			oldObj: func() storage.CSINode {
				obj := mkValidCSINodeDriverNode()
				obj.ResourceVersion = "1"
				return obj
			}(),
			updateObj: func() storage.CSINode {
				obj := mkValidCSINodeDriverNode(func(d *storage.CSINodeDriver) {
					d.Name = "io.kubernetes.storage.csi.other-driver"
				})
				obj.ResourceVersion = "1"
				return obj
			}(),
			expectedErrs: field.ErrorList{
				field.Forbidden(driverPath.Child("name"), "updates to driver name are forbidden"),
			},
		},
		"invalid update (driver nodeID changed)": {
			oldObj: func() storage.CSINode {
				obj := mkValidCSINodeDriverNode(func(d *storage.CSINodeDriver) {
					d.NodeID = "node-1"
				})
				obj.ResourceVersion = "1"
				return obj
			}(),
			updateObj: func() storage.CSINode {
				obj := mkValidCSINodeDriverNode(func(d *storage.CSINodeDriver) {
					d.NodeID = "node-2"
				})
				obj.ResourceVersion = "1"
				return obj
			}(),
			expectedErrs: field.ErrorList{
				field.Forbidden(driverPath.Child("nodeID"), "updates to driver nodeID are forbidden"),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx := genericapirequest.WithRequestInfo(
				genericapirequest.NewDefaultContext(),
				&genericapirequest.RequestInfo{
					APIPrefix:         "apis",
					APIGroup:          "storage.k8s.io",
					APIVersion:        apiVersion,
					Resource:          "csinodes",
					IsResourceRequest: true,
					Verb:              "update",
				},
			)

			apitesting.VerifyUpdateValidationEquivalence(
				t,
				ctx,
				&tc.updateObj,
				&tc.oldObj,
				Strategy.ValidateUpdate,
				tc.expectedErrs,
			)
		})
	}
}

func mkValidCSINodeDriverNode(tweaks ...func(*storage.CSINodeDriver)) storage.CSINode {
	driver := storage.CSINodeDriver{
		Name:   "io.kubernetes.storage.csi.driver-1",
		NodeID: "node-1",
	}
	for _, tweak := range tweaks {
		tweak(&driver)
	}

	return storage.CSINode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-1",
		},
		Spec: storage.CSINodeSpec{
			Drivers: []storage.CSINodeDriver{driver},
		},
	}
}
