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

package cloud_test

import (
	csapi "github.com/apache/cloudstack-go/v2/cloudstack"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"sigs.k8s.io/cluster-api-provider-cloudstack/pkg/cloud"
	dummies "sigs.k8s.io/cluster-api-provider-cloudstack/test/dummies/v1beta2"
)

var _ = Describe("Tag Unit Tests", func() {
	const (
		errorMessage = "Error"
	)

	fakeError := errors.New(errorMessage)
	var ( // Declare shared vars.
		mockCtrl   *gomock.Controller
		mockClient *csapi.CloudStackClient
		rs         *csapi.MockResourcetagsServiceIface
		client     cloud.Client
	)

	BeforeEach(func() {
		dummies.SetDummyVars()
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = csapi.NewMockClient(mockCtrl)
		rs = mockClient.Resourcetags.(*csapi.MockResourcetagsServiceIface)
		client = cloud.NewClientFromCSAPIClient(mockClient)
	})

	Context("Tag Integ Tests", Label("integ"), func() {
		BeforeEach(func() {
			client = realCloudClient
			FetchIntegTestResources()

			existingTags, err := client.GetTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)
			if err != nil {
				Fail("Failed to get existing tags. Error: " + err.Error())
			}
			if len(existingTags) > 0 {
				err = client.DeleteTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, existingTags)
				if err != nil {
					Fail("Failed to delete existing tags. Error: " + err.Error())
				}
			}
		})

		It("adds and gets a resource tag", func() {
			Ω(client.AddTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.Tags)).Should(Succeed())
			Ω(client.GetTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).Should(Equal(dummies.Tags))
		})

		It("deletes a resource tag", func() {
			Ω(client.AddTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.Tags)).Should(Succeed())
			Ω(client.DeleteTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.Tags)).Should(Succeed())
			Ω(client.GetTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).Should(Equal(map[string]string{}))
		})

		It("returns an error when you delete a tag that doesn't exist", func() {
			Ω(client.DeleteTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.Tags)).Should(Succeed())
		})

		It("adds the tags for a cluster (resource created by CAPC)", func() {
			Ω(client.AddCreatedByCAPCTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).
				Should(Succeed())
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).
				Should(Succeed())

			// Verify tags
			tags, err := client.GetTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(tags[dummies.CSClusterTagKey]).Should(Equal(dummies.CSClusterTagVal))
		})

		It("does not fail when the cluster tags are added twice", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())
		})

		It("doesn't adds the tags for a cluster (resource NOT created by CAPC)", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())

			// Verify tags
			tags, err := client.GetTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)
			Ω(err).Should(BeNil())
			Ω(tags[dummies.CreatedByCapcKey]).Should(Equal(""))
			Ω(tags[dummies.CSClusterTagKey]).Should(Equal(""))
		})

		It("deletes a cluster tag", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())
			Ω(client.DeleteClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())

			Ω(client.GetTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).ShouldNot(HaveKey(dummies.CSClusterTagKey))
		})

		It("adds and deletes a created by capc tag", func() {
			Ω(client.AddCreatedByCAPCTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).Should(Succeed())
			Ω(client.DeleteCreatedByCAPCTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).Should(Succeed())
		})

		It("does not fail when cluster and CAPC created tags are deleted twice", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())
			Ω(client.DeleteClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())
			Ω(client.DeleteClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())
			Ω(client.DeleteCreatedByCAPCTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).Should(Succeed())
			Ω(client.DeleteCreatedByCAPCTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).Should(Succeed())
		})

		It("does not allow a resource to be deleted when there are no tags", func() {
			tagsAllowDisposal, err := client.DoClusterTagsAllowDisposal(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)
			Ω(err).Should(BeNil())
			Ω(tagsAllowDisposal).Should(BeFalse())
		})

		It("does not allow a resource to be deleted when there is a cluster tag", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())
			tagsAllowDisposal, err := client.DoClusterTagsAllowDisposal(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)
			Ω(err).Should(BeNil())
			Ω(tagsAllowDisposal).Should(BeFalse())
		})

		It("does allow a resource to be deleted when there are no cluster tags and there is a CAPC created tag", func() {
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())
			Ω(client.AddCreatedByCAPCTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)).Should(Succeed())
			Ω(client.DeleteClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())

			tagsAllowDisposal, err := client.DoClusterTagsAllowDisposal(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID)
			Ω(err).Should(BeNil())
			Ω(tagsAllowDisposal).Should(BeTrue())
		})
	})

	Context("Add cluster tag", func() {
		It("Add cluster tag if managed by CAPC", func() {
			createdByCAPCResponse := &csapi.ListTagsResponse{Tags: []*csapi.Tag{{Key: cloud.CreatedByCAPCTagName, Value: "1"}}}
			rtlp := &csapi.ListTagsParams{}
			ctp := &csapi.CreateTagsParams{}
			rs.EXPECT().NewListTagsParams().Return(rtlp)
			rs.EXPECT().ListTags(rtlp).Return(createdByCAPCResponse, nil)
			rs.EXPECT().NewCreateTagsParams(gomock.Any(), gomock.Any(), gomock.Any()).Return(ctp)
			rs.EXPECT().CreateTags(ctp).Return(&csapi.CreateTagsResponse{}, nil)
			Ω(client.AddClusterTag(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, dummies.CSCluster)).Should(Succeed())
		})
	})

	Context("Delete tags", func() {
		It("delete resource tags fails", func() {
			tags := map[string]string{
				"key1": "value1",
			}
			rp := &csapi.DeleteTagsParams{}
			rs.EXPECT().NewDeleteTagsParams(gomock.Any(), gomock.Any()).Return(rp)
			rs.EXPECT().DeleteTags(rp).Return(nil, fakeError)
			rs.EXPECT().NewListTagsParams().Return(&csapi.ListTagsParams{})
			rs.EXPECT().ListTags(gomock.Any()).Return(&csapi.ListTagsResponse{
				Count: len(tags),
				Tags: []*csapi.Tag{{
					Key:   "key1",
					Value: "value1",
				}},
			}, nil)

			err := client.DeleteTags(cloud.ResourceTypeNetwork, dummies.CSISONet1.Spec.ID, tags)
			Ω(err).ShouldNot(Succeed())
			Ω(err.Error()).Should(ContainSubstring("could not remove tag"))
		})
	})

	Context("Get tags for a resource", func() {
		It("listing tags for a resource fails", func() {
			rs.EXPECT().NewListTagsParams().Return(&csapi.ListTagsParams{})
			rs.EXPECT().ListTags(gomock.Any()).Return(nil, fakeError)

			_, err := client.GetTags(cloud.ResourceTypeNetwork, dummies.ISONet1.ID)
			Ω(err).ShouldNot(Succeed())
		})
	})
})
