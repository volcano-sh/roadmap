# Hierarchy Queue

## Motivation

The hierarchy queue is derived from the queue management mechanism of YARN, refer to: https://blog.cloudera.com/yarn-capacity-scheduler/.

It is designed with multiple levels of resource constraints to better support fairshare scheduling in Volcano.

## Function Specification

The hierarchy queue has two important concepts: parent queues and leaf queues.

1. Parent queues do not themselves accept any application submissions directly. And they can contain more parent queues or leaf queues.
2. Leaf queues are the queues that live under a parent queue and accept applications. Leaf queues do not have any child queues.

### API

```go
type QueueSpec struct {
    ...
    // ParentQueue indicates who its parent queue is
    ParentQueue string `json:"parentQueue,omitempty" protobuf:"bytes,4,opt,name=parentQueue"`

    // ChildQueues indicates which child queues it has
    ChildQueues []string `json:"childQueues,omitempty" protobuf:"bytes,5,opt,name=childQueues"`

    // MinUserLimitPercent indicates the queue enforces a limit on the percentage of resources allocated to a job at any given time
    MinUserLimitPercent int `json:"minUserLimitPercent,omitempty" protobuf:"bytes,6,opt,name=minUserLimitPercent"`

    // UserLimitFactor indicates the multiple of the queue capacity which can be configured to allow a single user to acquire more resources
    UserLimitFactor float32 `json:"userLimitFactor,omitempty" protobuf:"bytes,7,opt,name=userLimitFactor"`
}
```

### Use cases

#### One: Resource sharing among multiple organizations

Suppose we now have two organizations (orgA, orgB), four horizontal queues (queue1, queue2, queue3, queue4). Queue1 and queue2 belong to orgA; queue3 and queue4 belong to orgB. They all weight 1.
If at first only queue1/queue2/queue3 were carving up and occupying the cluster resources, and queue1/queue2/queue3 each get 33% of the cluster resources.

```
queue1(33% --> 25%), queue2(33% --> 25%), queue3(33% --> 25%), queue4(0% --> 25%)
```

When queue4 wants to get resources, the queues' quota changes: they are all allocated 25%.
But if queue1 and queue2 can't be reclaimed, and queue3 can't reclaim enough resources to run the job in queue4, then queue4 has to wait for queue1/queue2/queue3's job to complete and release the resources. This can cause queue4 jobs to wait too long.

If we use hierarchy queue and orgA/orgB all weight 1 too, they will initially allocate 50% each, and their child queues: queues1 and queue2 get 25% of the resources and queue3 gets 50% of the resources.
When queue4 wants to get resources, more resources can be reclaimed from queue3, which makes it more likely the job requirements in queue4 will be met.
```
root
|--orgA(50%)
|  |--queue1(25%)
|  |--queue2(25%)
|--orgB(50%)
|  |--queue3(50% --> 25%)
|  |--queue4( 0% --> 25%)
```

#### Two: Resource sharing for multiple users in a single queue

To prevent a single user from monopolizing the resources of the entire queue, we use `MinUserLimitPercent` and `UserLimitFactor` to limit resource usage for users.

`MinUserLimitPercent`indicates the queue enforces a limit on the percentage of resources allocated to a job at any given time. The user limit can vary between a minimum and maximum value. The minimum value is set to this property value, and the maximum value depends on the number of users who have submitted applications.
For example, suppose the value of this filed is 25. If two users have submitted applications to a queue, no single user can use more than 50% of the queue resources. If a third user submits an application, no single user can use more than 33% of the queue resources. With 4 or more users, no user can use more than 25% of the queues resources. A value of 100 implies no user limits are imposed. The default is 100.

`UserLimitFactor` is set as a multiple of the queues minimum capacity where its value of 1 means the user can consume the entire minimum capacity of the queue. If `UserLimitFactor` is greater than 1 it's possible for the user to grow into maximum capacity and if the value is set to less than 1, such as 0.5, a user will only be able to obtain half of the minimum capacity of the queue. 

### Features of hierarchy queue

1. `root` is a top-level parent queue which represents the cluster itself. All other queues are direct or indirect child queues of `root`.
2. Initially, we only have one queue called `default` which belongs to `root`.
3. `ParentQueue` is optional. If its value is null when you create a queue, it defaults to `root` as its parent queue.
4. `ChildQueues` is optional. If its value is null when you create a queue, the queue will be a leaf queue.

### QueueController

1. Watching `PodGroup/Job` for status; The total number of `PodGroup`/`Job` owned by child queues is the number of `PodGroup`/`Job` owned by the parent queue.
2. If queue was deleted, also delete all related `PodGroup/Job` and child queues of this queue.
3. If queue was deleted and its parent queue will become a leaf queue, its parent queue's `ChildQueues` will change and its `MinUserLimitPercent`/`UserLimitFactor` will be valid.
4. If a parent queue is closed, its child queues will be closed too.

### Admission Controller

The admission controller will check `PodGroup`/`Job` 's queue when creation:

1. If the queue is parent queue, the creation will be rejected.
2. If no queue is specified, it will be scheduled to the `root.default` queue.

The admission controller will check other related queues when the queue creates:

1. If this queue's `ParentQueue` is a leaf queue, and it has some running jobs, the creation will be rejected.
2. If this queue meets the creation requirements, and its `ParentQueue` is a leaf queue. Its parent queue's `ChildQueues` will change and its `MinJobLimitPercent`/`JobLimitFactor` will be set to null.

### Cli

1. Because child queue's resource allocation depends on its parent queue's. We need to understand the hierarchy of each queue so that the administrator can clearly assign weights to queues.
2. We need to know who leaf queue is, because just leaf queue can accept applications.

So when we use `vqueues` to list all queues of the cluster, we can just use a display similar to `tree` command. Look at the following example.

```shell
$ vqueues
Name             Weight  State  Inqueue  Pending  Running  Unknown
root                  1   Open        6        0        6        0
|--default            5   Open        3        0        3        0
|--dev                5   Open        3        0        3        0
|  |--test1           1   Open        1        0        1        0
|  |--test2           2   Open        2        0        2        0 
```

## How to create?

You can create hierarchy queues one by one with yaml. Or you can use `annotations` to define hierarchy queue. Here is an example for creating a leaf queue and its parent queues.

```yaml
apiVersion: scheduling.volcano.sh/v1beta1
kind: Queue
metadata:
  name: test1
  annotations:
    "volcano.sh/hierarchy-queue": "root/dev/test1"
    "volcano.sh/hierarchy-queue-weights": "1/5/2"
    "volcano.sh/hierarchy-queue-reclaim": "true/true/false"
spec:
  weight: 1
...
```

Creation tips:
1. "volcano.sh/hierarchy-queue" will create as many queues as it defines, and they are a top-down parent-child relationship.
2. "volcano.sh/hierarchy-queue-weights" and "volcano.sh/hierarchy-queue-reclaim" must correspond one by one to the queue in "volcano.sh/hierarchy-queue".
3. If "volcano.sh/hierarchy-queue" contains a queue that has been created, it will just change his `ChildQueues` or `weight`, such as `root`.
4. If we also define the filed in `spec`, it will override the value in the `annotations` like `weight`.

## Conflict

This feature depends on the proportion plugin. If we use hierarchy queue, the hdrf should be disabled.