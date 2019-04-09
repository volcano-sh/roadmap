/*
Copyright 2019 The Volcano Authors.

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

package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"volcano.sh/volcano/pkg/apis/batch/v1alpha1"
)

var _ = Describe("Job E2E Test: Test Admission service", func() {
	cleanupResources := CleanupResources{}
	var context *context

	BeforeEach(func() {
		context = gContext
	})

	AfterEach(func() {
		deleteResources(gContext, cleanupResources)
	})

	It("Duplicated Task Name", func() {
		jobName := "job-duplicated"
		cleanupResources.Jobs = []string{jobName}

		_, err := createJobInner(context, &jobSpec{
			name: jobName,
			tasks: []taskSpec{
				{
					img:  defaultNginxImage,
					req:  oneCPU,
					min:  1,
					rep:  1,
					name: "duplicated",
				},
				{
					img:  defaultNginxImage,
					req:  oneCPU,
					min:  1,
					rep:  1,
					name: "duplicated",
				},
			},
		})
		Expect(err).To(HaveOccurred())
		stError, ok := err.(*errors.StatusError)
		Expect(ok).To(Equal(true))
		Expect(stError.ErrStatus.Code).To(Equal(int32(500)))
		Expect(stError.ErrStatus.Message).To(ContainSubstring("duplicated task name"))
	})

	It("Duplicated Policy Event", func() {
		jobName := "job-policy-duplicated"
		cleanupResources.Jobs = []string{jobName}

		_, err := createJobInner(context, &jobSpec{
			name: jobName,
			tasks: []taskSpec{
				{
					img:  defaultNginxImage,
					req:  oneCPU,
					min:  1,
					rep:  1,
					name: "taskname",
				},
			},
			policies: []v1alpha1.LifecyclePolicy{
				{
					Event:  v1alpha1.PodFailedEvent,
					Action: v1alpha1.AbortJobAction,
				},
				{
					Event:  v1alpha1.PodFailedEvent,
					Action: v1alpha1.RestartJobAction,
				},
			},
		})
		Expect(err).To(HaveOccurred())
		stError, ok := err.(*errors.StatusError)
		Expect(ok).To(Equal(true))
		Expect(stError.ErrStatus.Code).To(Equal(int32(500)))
		Expect(stError.ErrStatus.Message).To(ContainSubstring("duplicated job event policies"))
	})

	It("Min Available illegal", func() {
		jobName := "job-min-illegal"
		cleanupResources.Jobs = []string{jobName}

		_, err := createJobInner(context, &jobSpec{
			min:  2,
			name: jobName,
			tasks: []taskSpec{
				{
					img:  defaultNginxImage,
					req:  oneCPU,
					min:  1,
					rep:  1,
					name: "taskname",
				},
			},
		})
		Expect(err).To(HaveOccurred())
		stError, ok := err.(*errors.StatusError)
		Expect(ok).To(Equal(true))
		Expect(stError.ErrStatus.Code).To(Equal(int32(500)))
		Expect(stError.ErrStatus.Message).To(ContainSubstring("'minAvailable' should not be greater than total replicas in tasks"))
	})

	It("Job Plugin illegal", func() {
		jobName := "job-plugin-illegal"
		cleanupResources.Jobs = []string{jobName}

		_, err := createJobInner(context, &jobSpec{
			min:  1,
			name: jobName,
			plugins: map[string][]string{
				"big_plugin": {},
			},
			tasks: []taskSpec{
				{
					img:  defaultNginxImage,
					req:  oneCPU,
					min:  1,
					rep:  1,
					name: "taskname",
				},
			},
		})
		Expect(err).To(HaveOccurred())
		stError, ok := err.(*errors.StatusError)
		Expect(ok).To(Equal(true))
		Expect(stError.ErrStatus.Code).To(Equal(int32(500)))
		Expect(stError.ErrStatus.Message).To(ContainSubstring("unable to find job plugin: big_plugin"))
	})
})
