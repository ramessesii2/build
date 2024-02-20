// Copyright The Shipwright Contributors
//
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/shipwright-io/build/pkg/apis/build/v1beta1"
	test "github.com/shipwright-io/build/test/v1beta1_samples"
)

var _ = Describe("Integration tests BuildRuns and Service-accounts", func() {

	var (
		cbsObject      *v1beta1.ClusterBuildStrategy
		buildObject    *v1beta1.Build
		buildRunObject *v1beta1.BuildRun
		buildSample    []byte
		buildRunSample []byte
	)

	// Load the ClusterBuildStrategies before each test case
	BeforeEach(func() {
		cbsObject, err = tb.Catalog.LoadCBSWithName(STRATEGY+tb.Namespace, []byte(test.ClusterBuildStrategySingleStep))
		Expect(err).To(BeNil())

		err = tb.CreateClusterBuildStrategy(cbsObject)
		Expect(err).To(BeNil())
	})

	// Delete the ClusterBuildStrategies after each test case
	AfterEach(func() {

		_, err = tb.GetBuild(buildObject.Name)
		if err == nil {
			Expect(tb.DeleteBuild(buildObject.Name)).To(BeNil())
		}

		err := tb.DeleteClusterBuildStrategy(cbsObject.Name)
		Expect(err).To(BeNil())
	})

	// Override the Builds and BuildRuns CRDs instances to use
	// before an It() statement is executed
	JustBeforeEach(func() {
		if buildSample != nil {
			buildObject, err = tb.Catalog.LoadBuildWithNameAndStrategy(BUILD+tb.Namespace, STRATEGY+tb.Namespace, buildSample)
			Expect(err).To(BeNil())
		}

		if buildRunSample != nil {
			buildRunObject, err = tb.Catalog.LoadBRWithNameAndRef(BUILDRUN+tb.Namespace, BUILD+tb.Namespace, buildRunSample)
			Expect(err).To(BeNil())
		}
	})

	Context("when a buildrun is created with autogenerated service-account", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSWithShortTimeOutAndRefOutputSecret)
			buildRunSample = []byte(test.MinimalBuildRunWithSAGeneration)
		})

		It("creates a new service-account and deletes it after the build is terminated", func() {

			// loop and find a match
			var contains = func(secretList []corev1.ObjectReference, secretName string) bool {
				for _, s := range secretList {
					if s.Name == secretName {
						return true
					}
				}
				return false
			}

			sampleSecret := tb.Catalog.SecretWithAnnotation(*buildObject.Spec.Output.PushSecret, buildObject.Namespace)

			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			sa, err := tb.GetSA(buildRunObject.Name)
			Expect(err).To(BeNil())

			// Verify that the sa have our Build specified secret
			Expect(contains(sa.Secrets, *buildObject.Spec.Output.PushSecret)).To(BeTrue())

			_, err = tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())

			_, err = tb.GetSA(buildRunObject.Name)
			Expect(err).ToNot(BeNil())

		})
	})

	Context("when a buildrun is cancelled with an autogenerated serviceaccount", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSWithShortTimeOutAndRefOutputSecret)
			buildRunSample = []byte(test.MinimalBuildRunWithSAGeneration)
		})

		It("it deletes it after the build is cancelled", func() {

			// loop and find a match
			var contains = func(secretList []corev1.ObjectReference, secretName string) bool {
				for _, s := range secretList {
					if s.Name == secretName {
						return true
					}
				}
				return false
			}

			sampleSecret := tb.Catalog.SecretWithAnnotation(*buildObject.Spec.Output.PushSecret, buildObject.Namespace)

			Expect(tb.CreateSecret(sampleSecret)).To(BeNil())

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			bro, err := tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			sa, err := tb.GetSA(buildRunObject.Name)
			Expect(err).To(BeNil())

			// Verify that the sa have our Build specified secret
			Expect(contains(sa.Secrets, *buildObject.Spec.Output.PushSecret)).To(BeTrue())

			// cancel the br
			err = wait.PollImmediate(1*time.Second, 4*time.Second, func() (done bool, err error) {
				bro, err = tb.GetBRTillStartTime(buildRunObject.Name)
				if err != nil {
					GinkgoT().Logf("error on br get: %s\n", err.Error())
					return false, nil
				}

				bro.Spec.State = v1beta1.BuildRunRequestedStatePtr(v1beta1.BuildRunStateCancel)
				err = tb.UpdateBR(bro)
				if err != nil {
					GinkgoT().Logf("error on br update: %s\n", err.Error())
					return false, nil
				}
				return true, nil
			})
			Expect(err).To(BeNil())

			expectedReason := v1beta1.BuildRunStateCancel
			actualReason, err := tb.GetBRTillDesiredReason(buildRunObject.Name, expectedReason)
			Expect(err).To(BeNil(), fmt.Sprintf("failed to get desired BuildRun reason; expected %s, got %s", expectedReason, actualReason))

			// confirm complete
			_, err = tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())

			// confirm SA gone
			_, err = tb.GetSA(buildRunObject.Name)
			Expect(err).ToNot(BeNil())

		})
	})

	Context("when a buildrun is created without autogenerated service-account", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSWithShortTimeOut)
			buildRunSample = []byte(test.MinimalBuildRun)
		})

		It("uses the pipeline serviceaccount if exists", func() {
			Expect(tb.CreateSAFromName("pipeline")).To(BeNil())

			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(tr.Spec.ServiceAccountName).To(Equal("pipeline"))
		})

		It("defaults to default serviceaccount if pipeline serviceaccount is not specified", func() {
			expectedServiceAccount := "default"
			if _, err := tb.GetSA("pipeline"); err == nil {
				expectedServiceAccount = "pipeline"
			}
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			_, err = tb.GetBRTillStartTime(buildRunObject.Name)
			Expect(err).To(BeNil())

			tr, err := tb.GetTaskRunFromBuildRun(buildRunObject.Name)
			Expect(err).To(BeNil())
			Expect(tr.Spec.ServiceAccountName).To(Equal(expectedServiceAccount))
		})
	})

	Context("when a buildrun is created with a specified service-account", func() {

		BeforeEach(func() {
			buildSample = []byte(test.BuildCBSWithShortTimeOut)
			buildRunSample = []byte(test.MinimalBuildRunWithSpecifiedServiceAccount)
		})

		It("it fails and updates buildrun conditions if the specified serviceaccount doesn't exist", func() {
			Expect(tb.CreateBuild(buildObject)).To(BeNil())

			buildObject, err = tb.GetBuildTillValidation(buildObject.Name)
			Expect(err).To(BeNil())

			Expect(tb.CreateBR(buildRunObject)).To(BeNil())

			br, _ := tb.GetBRTillCompletion(buildRunObject.Name)
			Expect(err).To(BeNil())
			buildRunCondition := br.Status.GetCondition(v1beta1.Succeeded)

			Expect(buildRunCondition).ToNot(BeNil())
			Expect(buildRunCondition.Status).To(Equal(corev1.ConditionFalse))
			Expect(buildRunCondition.Reason).To(Equal("ServiceAccountNotFound"))
			Expect(buildRunCondition.Message).To(ContainSubstring("not found"))
		})
	})
})
