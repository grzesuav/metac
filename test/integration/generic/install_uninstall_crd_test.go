/*
Copyright 2019 The MayaData Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package generic

import (
	"testing"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/klog"

	"openebs.io/metac/apis/metacontroller/v1alpha1"
	"openebs.io/metac/controller/generic"
	"openebs.io/metac/test/integration/framework"
)

// TestInstallUninstallCRD will verify if GenericController can be
// used to implement install & uninstall of CustomResourceDefinition
// given install & uninstall of a namespace.
//
// This function will try to get a CRD installed when a target namespace
// gets installed. GenericController should also automatically uninstall
// this CRD when this target namespace is deleted.
func TestInstallUninstallCRD(t *testing.T) {
	// if testing.Short() {
	// 	t.Skip("Skipping TestApplyDeleteCRD in short mode")
	// }

	// namespace to setup GenericController
	ctlNSNamePrefix := "gctl-test"

	// name of the GenericController
	ctlName := "install-un-crd-ctrl"

	// name of the target namespace which is watched by GenericController
	targetNSName := "install-un-ns"

	// name of the target CRD which is reconciled by GenericController
	targetCRDName := "storages.dao.amitd.io"

	f := framework.NewFixture(t)
	defer f.TearDown()

	// create namespace to setup GenericController resources
	ctlNS := f.CreateNamespaceGen(ctlNSNamePrefix)

	// -------------------------------------------------------------------------
	// Define the "reconcile logic" for sync i.e. create/update events of watch
	// -------------------------------------------------------------------------
	//
	// NOTE:
	// 	Sync ensures creation of target CRD via attachments
	sHook := f.ServeWebhook(func(body []byte) ([]byte, error) {
		req := generic.SyncHookRequest{}
		if uerr := json.Unmarshal(body, &req); uerr != nil {
			return nil, uerr
		}

		// initialize the hook response
		resp := generic.SyncHookResponse{}

		// we desire this CRD object
		crd := framework.BuildUnstructuredObjFromJSON(
			"apiextensions.k8s.io/v1beta1",
			"CustomResourceDefinition",
			targetCRDName,
			`{
				"spec": {
					"group": "dao.amitd.io",
					"version": "v1alpha1",
					"scope": "Namespaced",
					"names": {
						"plural": "storages",
						"singular": "storage",
						"kind": "Storage",
						"shortNames": ["stor"]
					},
					"additionalPrinterColumns": [
						{
							"JSONPath": ".spec.capacity",
							"name": "Capacity",
							"description": "Capacity of the storage",
							"type": "string"
						},
						{
							"JSONPath": ".spec.nodeName",
							"name": "NodeName",
							"description": "Node where the storage gets attached",
							"type": "string"
						},
						{
							"JSONPath": ".status.phase",
							"name": "Status",
							"description": "Identifies the current status of the storage",
							"type": "string"
						}
					]
				}
			}`,
		)

		// add CRD to attachments to let GenericController
		// sync i.e. create
		resp.Attachments = append(resp.Attachments, crd)

		return json.Marshal(resp)
	})

	// ---------------------------------------------------------------------
	// Define the "reconcile logic" for finalize i.e. delete event of watch
	// ---------------------------------------------------------------------
	//
	// NOTE:
	// 	Finalize ensures deletion of target CRD via attachments

	// isFinalized helps in determining if the CRD was deleted
	// and is no longer a part of attachment.
	var isFinalized bool

	fHook := f.ServeWebhook(func(body []byte) ([]byte, error) {
		req := generic.SyncHookRequest{}
		if uerr := json.Unmarshal(body, &req); uerr != nil {
			return nil, uerr
		}

		// initialize the hook response
		resp := generic.SyncHookResponse{}

		if req.Watch.GetDeletionTimestamp() == nil {
			resp.ResyncAfterSeconds = 2

			// no need to reconcile the attachments since
			// watch is not marked for deletion
			resp.SkipReconcile = true
		} else {
			// set attachments to nil to let GenericController
			// finalize i.e. delete CRD
			resp.Attachments = nil

			// finalize hook should be executed till its request
			// has attachments
			if req.Attachments.IsEmpty() {
				// since all attachments are deleted from cluster
				// indicate GenericController to mark completion
				// of finalize hook
				resp.Finalized = true
			} else {
				// if there are still attachments seen in the request
				// keep resyncing the watch
				resp.ResyncAfterSeconds = 2
			}
		}

		// Set this to help in verifing this test case outside
		// of this finalize block
		isFinalized = resp.Finalized

		klog.V(2).Infof("Finalize: Req.Attachments.Len=%d", req.Attachments.Len())
		return json.Marshal(resp)
	})

	// ---------------------------------------------------------
	// Define & Apply a GenericController i.e. a Meta Controller
	// ---------------------------------------------------------

	// This is one of the meta controller that is defined as
	// a Kubernetes custom resource. It listens to the resource
	// specified in the watch field and acts against the resources
	// specified in the attachments field.
	f.CreateGenericController(
		ctlName,
		ctlNS.Name,

		// set 'sync' as well as 'finalize' hooks
		generic.WithWebhookSyncURL(&sHook.URL),
		generic.WithWebhookFinalizeURL(&fHook.URL),

		// Namespace is the watched resource
		generic.WithWatch(
			&v1alpha1.GenericControllerResource{
				ResourceRule: v1alpha1.ResourceRule{
					APIVersion: "v1",
					Resource:   "namespaces",
				},
				// We are interested only for the target namespace only
				NameSelector: []string{targetNSName},
			},
		),

		// CRDs are the attachments
		//
		// This is done so as to implement create & delete of CRD
		// when above watch resource i.e. namespce is created & deleted.
		generic.WithAttachments(
			[]*v1alpha1.GenericControllerAttachment{
				// We want the target CRD only i.e. storages.dao.amitd.io
				&v1alpha1.GenericControllerAttachment{
					GenericControllerResource: v1alpha1.GenericControllerResource{
						ResourceRule: v1alpha1.ResourceRule{
							APIVersion: "apiextensions.k8s.io/v1beta1",
							Resource:   "customresourcedefinitions",
						},
						NameSelector: []string{targetCRDName},
					},
				},
			},
		),
	)

	var err error

	// ---------------------------------------------------
	// Create the target namespace i.e. target under test
	// ---------------------------------------------------
	//
	// NOTE:
	// 	This triggers reconciliation
	_, err = f.GetTypedClientset().
		CoreV1().
		Namespaces().
		Create(
			&v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: targetNSName,
				},
			},
		)
	if err != nil {
		t.Fatal(err)
	}

	// Need to wait & see if our controller works as expected
	// Make sure the specified attachments i.e. CRD is created
	klog.Infof("Wait for creation of CRD %s", targetCRDName)

	crdCreateErr := f.Wait(func() (bool, error) {
		// ------------------------------------------------
		// verify if target CRD is created i.e. reconciled
		// ------------------------------------------------
		crdCreateObj, createErr := f.GetCRDClient().
			CustomResourceDefinitions().
			Get(
				targetCRDName,
				metav1.GetOptions{},
			)
		if createErr != nil {
			return false, createErr
		}

		if crdCreateObj == nil {
			return false, errors.Errorf(
				"CRD %s is not created",
				targetCRDName,
			)
		}

		// condition passed
		return true, nil
	})
	if crdCreateErr != nil {
		t.Fatalf("CRD %s wasn't created: %v", targetCRDName, crdCreateErr)
	}

	// Wait till target namespace is assigned with GenericController's
	// finalizer
	//
	// NOTE:
	//	GenericController automatically updates the watch with
	// its own finalizer if it finds a finalize hook in its
	// specifications.
	nsWithFErr := f.Wait(func() (bool, error) {
		nsWithF, err := f.GetTypedClientset().
			CoreV1().
			Namespaces().
			Get(
				targetNSName,
				metav1.GetOptions{},
			)
		if err != nil {
			return false, err
		}
		for _, finalizer := range nsWithF.GetFinalizers() {
			if finalizer == "protect.gctl.metac.openebs.io/"+ctlNS.GetName()+"-"+ctlName {
				return true, nil
			}
		}
		return false, errors.Errorf(
			"Namespace %s is not set with gctl finalizer",
			targetNSName,
		)
	})
	if nsWithFErr != nil {
		// we wait till timeout & panic if condition is not met
		t.Fatal(nsWithFErr)
	}

	// ------------------------------------------------------
	// Trigger finalize by deleting the target namespace
	// ------------------------------------------------------

	err = f.GetTypedClientset().
		CoreV1().
		Namespaces().
		Delete(
			targetNSName,
			&metav1.DeleteOptions{},
		)
	if err != nil {
		t.Fatal(err)
	}

	// Need to wait & see if our controller works as expected
	// Make sure the specified attachments i.e. CRD is deleted
	klog.Infof("Wait for deletion of CRD %s", targetCRDName)

	crdDelErr := f.Wait(func() (bool, error) {
		var getErr error

		// ------------------------------------------------
		// verify if target CRD is deleted i.e. reconciled
		// ------------------------------------------------

		if isFinalized {
			return true, nil
		}

		crdObj, getErr := f.GetCRDClient().
			CustomResourceDefinitions().
			Get(
				targetCRDName,
				metav1.GetOptions{},
			)

		if getErr != nil && !apierrors.IsNotFound(getErr) {
			return false, getErr
		}

		if crdObj != nil && crdObj.GetDeletionTimestamp() == nil {
			return false, errors.Errorf(
				"CRD %s is not marked for deletion",
				targetCRDName,
			)
		}

		// condition passed
		return true, nil
	})

	if crdDelErr != nil {
		t.Fatalf("CRD %s wasn't deleted: %v", targetCRDName, crdDelErr)
	}

	klog.Infof("Test Install Uninstall CRD %s passed", targetCRDName)
}
