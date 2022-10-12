/*
Copyright 2022 The Kubernetes Authors.

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

package utils

import (
	"fmt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"strings"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	infrav1 "sigs.k8s.io/cluster-api-provider-cloudstack/api/v1beta2"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetOrCreateAffinityGroup of the passed name that's owned by the failure domain of the reconciliation subject and
// the control plane that manages it.
func (r *ReconciliationRunner) GetOrCreateAffinityGroup(
	name string,
	affinityType string,
	ag *infrav1.CloudStackAffinityGroup,
	fd *infrav1.CloudStackFailureDomain) CloudStackReconcilerMethod {

	return func() (ctrl.Result, error) {
		// Start by attempting a fetch.
		lowerName := strings.ToLower(name)
		namespace := r.ReconciliationSubject.GetNamespace()
		objKey := client.ObjectKey{Namespace: namespace, Name: lowerName}
		if err := r.K8sClient.Get(r.RequestCtx, objKey, ag); client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		} else if ag.Name != "" {
			return ctrl.Result{}, nil
		} // Didn't find a group, so create instead.

		// Set affinity group type.
		if affinityType == infrav1.ProAffinity {
			ag.Spec.Type = "host affinity"
		} else if affinityType == infrav1.AntiAffinity {
			ag.Spec.Type = "host anti-affinity"
		} else {
			return ctrl.Result{}, errors.Errorf("unrecognized affinity type %s", affinityType)
		}

		// Setup basic metadata.
		ag.Name = name
		ag.Spec.Name = name
		ag.ObjectMeta = r.NewChildObjectMeta(lowerName)

		// Replace owner reference with controller of CAPI and CloudStack machines and FailureDomain.
		for _, ref := range r.ReconciliationSubject.GetOwnerReferences() {
			if strings.EqualFold(ref.Kind, "EtcdadmCluster") ||
				strings.EqualFold(ref.Kind, "KubeadmControlPlane") ||
				strings.EqualFold(ref.Kind, "MachineSet") {
				ag.OwnerReferences = []metav1.OwnerReference{ref}
				break
			}
		}
		ag.OwnerReferences = append(ag.OwnerReferences,
			metav1.OwnerReference{
				Name:       fd.Name,
				Kind:       fd.Kind,
				APIVersion: fd.APIVersion,
				UID:        fd.UID,
			})

		if err := r.K8sClient.Create(r.RequestCtx, ag); err != nil && !ContainsAlreadyExistsSubstring(err) {
			return r.ReturnWrappedError(err, "creating affinity group CRD")
		}
		return ctrl.Result{}, nil
	}
}

// The computed affinity group name relevant to this machine.
func GenerateAffinityGroupName(csm infrav1.CloudStackMachine, capiMachine *clusterv1.Machine) (string, error) {
	managerOwnerRef := GetManagementOwnerRef(capiMachine)
	if managerOwnerRef == nil {
		return "", errors.Errorf("could not find owner UID for %s/%s", csm.Namespace, csm.Name)
	}
	titleCaser := cases.Title(language.English)
	return fmt.Sprintf("%sAffinity-%s-%s-%s",
		titleCaser.String(csm.Spec.Affinity), managerOwnerRef.Name, managerOwnerRef.UID, csm.Spec.FailureDomainName), nil
}
