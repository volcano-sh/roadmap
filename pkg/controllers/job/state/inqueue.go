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

package state

import (
	vkv1 "volcano.sh/volcano/pkg/apis/batch/v1alpha1"
	"volcano.sh/volcano/pkg/controllers/apis"
)

type inqueueState struct {
	job *apis.JobInfo
}

func (ps *inqueueState) Execute(action vkv1.Action) error {
	switch action {
	case vkv1.RestartJobAction:
		return KillJob(ps.job, PodRetainPhaseNone, func(status *vkv1.JobStatus) {
			UpdateJobPhase(status, vkv1.Restarting, "")
			status.RetryCount++
		})

	case vkv1.AbortJobAction:
		return KillJob(ps.job, PodRetainPhaseSoft, func(status *vkv1.JobStatus) {
			UpdateJobPhase(status, vkv1.Aborting, "")
		})
	case vkv1.CompleteJobAction:
		return KillJob(ps.job, PodRetainPhaseSoft, func(status *vkv1.JobStatus) {
			UpdateJobPhase(status, vkv1.Completing, "")
		})
	case vkv1.TerminateJobAction:
		return KillJob(ps.job, PodRetainPhaseSoft, func(status *vkv1.JobStatus) {
			UpdateJobPhase(status, vkv1.Terminating, "")
		})
	default:
		return SyncJob(ps.job, func(status *vkv1.JobStatus) {
			if ps.job.Job.Spec.MinAvailable <= status.Running+status.Succeeded+status.Failed {
				UpdateJobPhase(status, vkv1.Running, "")
			}
		})
	}
}
