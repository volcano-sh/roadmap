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

package cache

import (
	"fmt"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"

	schedulerv1beta1 "volcano.sh/volcano/pkg/apis/scheduling/v1beta1"
	"volcano.sh/volcano/pkg/scheduler/api"
)

func TestSchedulerCache_updateTask(t *testing.T) {
	namespace := "test"
	owner := buildOwnerReference("j1")

	tests := []struct {
		Name        string
		OldPod      *v1.Pod
		NewPod      *v1.Pod
		Nodes       []*v1.Node
		JobInfo     *api.JobInfo
		OldTaskInfo *api.TaskInfo
		NewTaskInfo *api.TaskInfo
		Expected    error
	}{
		{
			Name:   "Success Case",
			OldPod: buildPod(namespace, "p1", "n1", v1.PodRunning, buildResourceList("1000m", "1G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			NewPod: buildPod(namespace, "p1", "n1", v1.PodRunning, buildResourceList("1000m", "2G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			Nodes: []*v1.Node{
				buildNode("n1", buildResourceList("2000m", "10G")),
			},
			OldTaskInfo: &api.TaskInfo{},
			NewTaskInfo: &api.TaskInfo{},
			Expected:    nil,
		},
		{
			Name:   "Error Case",
			OldPod: buildPod(namespace, "p1", "n1", v1.PodSucceeded, buildResourceList("1000m", "1G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			NewPod: buildPod(namespace, "p1", "n1", v1.PodRunning, buildResourceList("1000m", "2G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			Nodes: []*v1.Node{
				buildNode("n1", buildResourceList("2000m", "10G")),
			},
			OldTaskInfo: &api.TaskInfo{},
			NewTaskInfo: &api.TaskInfo{},
			Expected:    fmt.Errorf("failed to find task <%s/%s> on host <%s>", namespace, "p1", "n1"),
		},
	}

	for i, test := range tests {
		cache := &SchedulerCache{
			Jobs:  make(map[api.JobID]*api.JobInfo),
			Nodes: make(map[string]*api.NodeInfo),
		}

		for _, n := range test.Nodes {
			cache.AddNode(n)
		}

		cache.AddPod(test.OldPod)
		test.OldTaskInfo = api.NewTaskInfo(test.OldPod)
		test.NewTaskInfo = api.NewTaskInfo(test.NewPod)

		new := cache.updateTask(test.OldTaskInfo, test.NewTaskInfo)

		if test.Expected != nil && new != nil && !strings.Contains(new.Error(), test.Expected.Error()) {
			t.Errorf("Expected Error to be %v but got %v in case %d", test.Expected, new, i)
		}
	}
}

func TestSchedulerCache_UpdatePod(t *testing.T) {
	namespace := "test"
	owner := buildOwnerReference("j1")

	tests := []struct {
		Name     string
		OldPod   *v1.Pod
		NewPod   *v1.Pod
		Nodes    []*v1.Node
		JobInfo  *api.JobInfo
		Expected error
	}{
		{
			Name:   "Success Case",
			OldPod: buildPod(namespace, "p1", "n1", v1.PodRunning, buildResourceList("1000m", "1G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			NewPod: buildPod(namespace, "p1", "n1", v1.PodRunning, buildResourceList("1000m", "2G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			Nodes: []*v1.Node{
				buildNode("n1", buildResourceList("2000m", "10G")),
			},
			Expected: nil,
		},
		{
			Name:   "Error Case",
			OldPod: buildPod(namespace, "p1", "n1", v1.PodSucceeded, buildResourceList("1000m", "1G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			NewPod: buildPod(namespace, "p1", "n1", v1.PodRunning, buildResourceList("1000m", "2G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			Nodes: []*v1.Node{
				buildNode("n1", buildResourceList("2000m", "10G")),
			},
			Expected: fmt.Errorf("failed to find task <%s/%s> on host <%s>", namespace, "p1", "n1"),
		},
	}

	for i, test := range tests {
		cache := &SchedulerCache{
			Jobs:  make(map[api.JobID]*api.JobInfo),
			Nodes: make(map[string]*api.NodeInfo),
		}

		for _, n := range test.Nodes {
			cache.AddNode(n)
		}

		cache.AddPod(test.OldPod)

		new := cache.updatePod(test.OldPod, test.NewPod)

		if test.Expected != nil && new != nil && !strings.Contains(new.Error(), test.Expected.Error()) {
			t.Errorf("Expected Error to be %v but got %v in case %d", test.Expected, new, i)
		}
	}
}

func TestSchedulerCache_AddPodGroupV1beta1(t *testing.T) {
	namespace := "test"
	owner := buildOwnerReference("j1")

	tests := []struct {
		Name     string
		Pod      *v1.Pod
		Nodes    []*v1.Node
		PodGroup interface{}
		Expected *schedulerv1beta1.PodGroup
	}{
		{
			Name: "Success Case",
			Pod:  buildPod(namespace, "p1", "n1", v1.PodRunning, buildResourceList("1000m", "1G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			Nodes: []*v1.Node{
				buildNode("n1", buildResourceList("2000m", "10G")),
			},
			PodGroup: &schedulerv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "j1",
					Namespace: namespace,
				},
			},
			Expected: &schedulerv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "j1",
					Namespace: namespace,
				},
			},
		},
		{
			Name: "Error Case: 1 - Wrong Type",
			Pod:  buildPod(namespace, "p1", "n1", v1.PodRunning, buildResourceList("1000m", "1G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			Nodes: []*v1.Node{
				buildNode("n1", buildResourceList("2000m", "10G")),
			},
			PodGroup: &schedulerv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "j1",
					Namespace: namespace,
				},
			},
			Expected: nil,
		},
		{
			Name: "Error Case: 2 - PodGroup Without Identity",
			Pod:  buildPod(namespace, "p1", "n1", v1.PodRunning, buildResourceList("1000m", "1G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			Nodes: []*v1.Node{
				buildNode("n1", buildResourceList("2000m", "10G")),
			},
			PodGroup: &schedulerv1beta1.PodGroup{
				Status: schedulerv1beta1.PodGroupStatus{
					Running: int32(1),
				},
			},
			Expected: nil,
		},
	}

	for i, test := range tests {
		cache := &SchedulerCache{
			Jobs:  make(map[api.JobID]*api.JobInfo),
			Nodes: make(map[string]*api.NodeInfo),
		}

		for _, n := range test.Nodes {
			cache.AddNode(n)
		}
		test.Pod.Annotations = map[string]string{
			"scheduling.k8s.io/group-name": "j1",
		}
		cache.AddPod(test.Pod)

		cache.AddPodGroup(test.PodGroup)
		jobID := api.JobID("test/j1")

		job := cache.Jobs[jobID]
		pg := job.PodGroup

		if test.Expected != nil && (pg.Namespace != test.Expected.Namespace || pg.Name != test.Expected.Name) {
			t.Errorf("Expected pg to be: %v but got :%v in case %d", test.Expected, pg, i)
		}
	}
}

func TestSchedulerCache_UpdatePodGroupV1beta1(t *testing.T) {
	namespace := "test"
	owner := buildOwnerReference("j1")

	tests := []struct {
		Name        string
		Pod         *v1.Pod
		Nodes       []*v1.Node
		OldPodGroup interface{}
		NewPodGroup interface{}
		Expected    *schedulerv1beta1.PodGroup
	}{
		{
			Name: "Success Case",
			Pod:  buildPod(namespace, "p1", "n1", v1.PodRunning, buildResourceList("1000m", "1G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			Nodes: []*v1.Node{
				buildNode("n1", buildResourceList("2000m", "10G")),
			},
			OldPodGroup: &schedulerv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "j1",
					Namespace: namespace,
				},
			},
			NewPodGroup: &schedulerv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "j1-updated",
					Namespace: namespace,
				},
			},
			Expected: &schedulerv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "j1-updated",
					Namespace: namespace,
				},
			},
		},
		{
			Name: "Error Case: 1 - Wrong Type(OldPodGroup)",
			Pod:  buildPod(namespace, "p1", "n1", v1.PodRunning, buildResourceList("1000m", "1G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			Nodes: []*v1.Node{
				buildNode("n1", buildResourceList("2000m", "10G")),
			},
			OldPodGroup: &schedulerv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "j1",
					Namespace: namespace,
				},
			},
			NewPodGroup: &schedulerv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "j1-updated",
					Namespace: namespace,
				},
			},
			Expected: nil,
		},
		{
			Name: "Error Case: 2 - PodGroup Without Identity",
			Pod:  buildPod(namespace, "p1", "n1", v1.PodRunning, buildResourceList("1000m", "1G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			Nodes: []*v1.Node{
				buildNode("n1", buildResourceList("2000m", "10G")),
			},
			NewPodGroup: &schedulerv1beta1.PodGroup{
				Status: schedulerv1beta1.PodGroupStatus{
					Running: int32(1),
				},
			},
			OldPodGroup: &schedulerv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "j1-updated",
					Namespace: namespace,
				},
			},
			Expected: nil,
		},
		{
			Name: "Error Case: 3 - Wrong Type(NewPodGroup)",
			Pod:  buildPod(namespace, "p1", "n1", v1.PodRunning, buildResourceList("1000m", "1G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			Nodes: []*v1.Node{
				buildNode("n1", buildResourceList("2000m", "10G")),
			},
			OldPodGroup: &schedulerv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "j1",
					Namespace: namespace,
				},
			},
			NewPodGroup: &schedulerv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "j1-updated",
					Namespace: namespace,
				},
			},
			Expected: nil,
		},
	}

	for i, test := range tests {
		cache := &SchedulerCache{
			Jobs:  make(map[api.JobID]*api.JobInfo),
			Nodes: make(map[string]*api.NodeInfo),
		}

		for _, n := range test.Nodes {
			cache.AddNode(n)
		}
		test.Pod.Annotations = map[string]string{
			"scheduling.k8s.io/group-name": "j1",
		}
		cache.AddPod(test.Pod)

		cache.UpdateQueue(test.OldPodGroup, test.NewPodGroup)
		jobID := api.JobID("test/j1")

		job := cache.Jobs[jobID]
		pg := job.PodGroup

		if test.Expected != nil && pg != nil && (pg.Namespace != test.Expected.Namespace || pg.Name != test.Expected.Name) {
			t.Errorf("Expected pg to be: %v but got :%v in case %d", test.Expected, pg, i)
		}
	}
}

func TestSchedulerCache_DeletePodGroupV1beta1(t *testing.T) {
	namespace := "test"
	owner := buildOwnerReference("j1")

	tests := []struct {
		Name     string
		Pod      *v1.Pod
		Nodes    []*v1.Node
		PodGroup interface{}
		Expected *schedulerv1beta1.PodGroup
	}{
		{
			Name: "Success Case",
			Pod:  buildPod(namespace, "p1", "n1", v1.PodRunning, buildResourceList("1000m", "1G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			Nodes: []*v1.Node{
				buildNode("n1", buildResourceList("2000m", "10G")),
			},
			PodGroup: &schedulerv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "j1",
					Namespace: namespace,
				},
			},
			Expected: nil,
		},
		{
			Name: "Error Case: 1 - Wrong Type",
			Pod:  buildPod(namespace, "p1", "n1", v1.PodRunning, buildResourceList("1000m", "1G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			Nodes: []*v1.Node{
				buildNode("n1", buildResourceList("2000m", "10G")),
			},
			PodGroup: &schedulerv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "j1",
					Namespace: namespace,
				},
			},
			Expected: &schedulerv1beta1.PodGroup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "j1",
					Namespace: namespace,
				},
			},
		},
		{
			Name: "Error Case: 2 - PodGroup Without Identity",
			Pod:  buildPod(namespace, "p1", "n1", v1.PodRunning, buildResourceList("1000m", "1G"), []metav1.OwnerReference{owner}, make(map[string]string)),
			Nodes: []*v1.Node{
				buildNode("n1", buildResourceList("2000m", "10G")),
			},
			PodGroup: &schedulerv1beta1.PodGroup{
				Status: schedulerv1beta1.PodGroupStatus{
					Running: int32(1),
				},
			},
			Expected: &schedulerv1beta1.PodGroup{
				Status: schedulerv1beta1.PodGroupStatus{
					Running: int32(1),
				},
			},
		},
	}

	for i, test := range tests {
		cache := &SchedulerCache{
			Jobs:  make(map[api.JobID]*api.JobInfo),
			Nodes: make(map[string]*api.NodeInfo),
		}

		cache.deletedJobs = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

		for _, n := range test.Nodes {
			cache.AddNode(n)
		}
		test.Pod.Annotations = map[string]string{
			"scheduling.k8s.io/group-name": "j1",
		}
		cache.AddPod(test.Pod)

		cache.AddPodGroup(test.PodGroup)

		cache.DeletePodGroup(test.PodGroup)
		jobID := api.JobID("test/j1")

		job := cache.Jobs[jobID]

		if test.Expected == nil && job.PodGroup != nil {
			t.Errorf("Expected job  to be: %v but got :%v in case %d", test.Expected, job, i)
		}
	}
}

func TestSchedulerCache_AddQueueV1beta1(t *testing.T) {
	namespace := "test"

	tests := []struct {
		Name     string
		Queue    interface{}
		Expected *schedulerv1beta1.Queue
	}{
		{
			Name: "Success Case",
			Queue: &schedulerv1beta1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "q1",
					Namespace: namespace,
				},
			},
			Expected: &schedulerv1beta1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "q1",
					Namespace: namespace,
				},
			},
		},
		{
			Name: "Error Case: 1 - Wrong Type",
			Queue: &schedulerv1beta1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "q1",
					Namespace: namespace,
				},
			},
			Expected: nil,
		},
	}

	for i, test := range tests {
		cache := &SchedulerCache{
			Jobs:   make(map[api.JobID]*api.JobInfo),
			Nodes:  make(map[string]*api.NodeInfo),
			Queues: make(map[api.QueueID]*api.QueueInfo)}

		cache.AddQueue(test.Queue)

		queue := cache.Queues["q1"]

		if test.Expected != nil && queue != nil && queue.Queue != nil && (queue.Queue.Namespace != test.Expected.Namespace || queue.Queue.Name != test.Expected.Name) {
			t.Errorf("Expected: %v but got: %v in case %d", test.Expected, queue.Queue, i)
		}
	}
}

func TestSchedulerCache_UpdateQueueV1beta1(t *testing.T) {
	namespace := "test"

	tests := []struct {
		Name     string
		OldQueue interface{}
		NewQueue interface{}
		Expected *schedulerv1beta1.Queue
	}{
		{
			Name: "Success Case",
			OldQueue: &schedulerv1beta1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "q1",
					Namespace: namespace,
				},
			},
			NewQueue: &schedulerv1beta1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "q1-updated",
					Namespace: namespace,
				},
			},
			Expected: &schedulerv1beta1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "q1-updated",
					Namespace: namespace,
				},
			},
		},
		{
			Name: "Error Case: 1 - Wrong Type(OldQueue)",
			OldQueue: &schedulerv1beta1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "q1",
					Namespace: namespace,
				},
			},
			NewQueue: &schedulerv1beta1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "q1-updated",
					Namespace: namespace,
				},
			},
			Expected: nil,
		},
		{
			Name: "Error Case: 2 - Wrong Type(NewQueue)",
			OldQueue: &schedulerv1beta1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "q1",
					Namespace: namespace,
				},
			},
			NewQueue: &schedulerv1beta1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "q1-updated",
					Namespace: namespace,
				},
			},
			Expected: nil,
		},
	}

	for i, test := range tests {
		cache := &SchedulerCache{
			Jobs:   make(map[api.JobID]*api.JobInfo),
			Nodes:  make(map[string]*api.NodeInfo),
			Queues: make(map[api.QueueID]*api.QueueInfo),
		}

		cache.UpdateQueue(test.OldQueue, test.NewQueue)

		queue := cache.Queues["q1-updated"]

		if test.Expected != nil && queue != nil && queue.Queue != nil && (queue.Queue.Namespace != test.Expected.Namespace || queue.Queue.Name != test.Expected.Name) {
			t.Errorf("Expected: %v but got: %v in case %d", test.Expected, queue.Queue, i)
		}
	}
}

func TestSchedulerCache_DeleteQueueV1beta1(t *testing.T) {
	namespace := "test"

	tests := []struct {
		Name     string
		Queue    interface{}
		Expected *schedulerv1beta1.Queue
	}{
		{
			Name: "Success Case",
			Queue: &schedulerv1beta1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "q1",
					Namespace: namespace,
				},
			},
			Expected: &schedulerv1beta1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "q1",
					Namespace: namespace,
				},
			},
		},
		{
			Name: "Error Case: 1 - Wrong Type",
			Queue: &schedulerv1beta1.Queue{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "q1",
					Namespace: namespace,
				},
			},
			Expected: nil,
		},
	}

	for i, test := range tests {
		cache := &SchedulerCache{
			Jobs:   make(map[api.JobID]*api.JobInfo),
			Nodes:  make(map[string]*api.NodeInfo),
			Queues: make(map[api.QueueID]*api.QueueInfo),
		}

		cache.AddQueue(test.Queue)
		cache.DeleteQueue(test.Queue)

		queue := cache.Queues["q1"]

		if test.Expected == nil && queue != nil {
			t.Errorf("Expected: %v but got: %v in case %d", test.Expected, queue, i)
		}

		if test.Expected != nil && queue != nil && queue.Queue != nil && (queue.Queue.Namespace != test.Expected.Namespace || queue.Queue.Name != test.Expected.Name) {
			t.Errorf("Expected: %v but got: %v in case %d", test.Expected, queue.Queue, i)
		}
	}
}
