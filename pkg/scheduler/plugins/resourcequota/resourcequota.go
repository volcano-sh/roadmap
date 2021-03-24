package resourcequota

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/quota/v1"

	"volcano.sh/volcano/pkg/apis/scheduling"
	"volcano.sh/volcano/pkg/scheduler/api"
	"volcano.sh/volcano/pkg/scheduler/framework"
	"volcano.sh/volcano/pkg/scheduler/plugins/util"
)

// PluginName indicates name of volcano scheduler plugin.
const PluginName = "resourcequota"

// resourceQuota scope not supported
type resourceQuotaPlugin struct {
	// Arguments given for the plugin
	pluginArguments framework.Arguments
}

// New return resourcequota plugin
func New(arguments framework.Arguments) framework.Plugin {
	return &resourceQuotaPlugin{
		pluginArguments: arguments,
	}
}

func (rq *resourceQuotaPlugin) Name() string {
	return PluginName
}

func (rq *resourceQuotaPlugin) OnSessionOpen(ssn *framework.Session) {
	ssn.AddJobEnqueueableFn(rq.Name(), func(obj interface{}) int {
		job := obj.(*api.JobInfo)

		resourcesRequests := job.PodGroup.Spec.MinQuotas
		if resourcesRequests == nil {
			klog.V(3).Infof("job: %s/%s minQuotas not exist, skip check resource quota", job.Namespace, job.Name)
			return util.Permit
		}

		quotas := ssn.NamespaceInfo[api.NamespaceName(job.Namespace)].RQStatus
		for _, resourceQuota := range quotas {
			hardResources := quota.ResourceNames(resourceQuota.Hard)
			requestedUsage := quota.Mask(*resourcesRequests, hardResources)
			newUsage := quota.Add(resourceQuota.Used, requestedUsage)
			maskedNewUsage := quota.Mask(newUsage, quota.ResourceNames(requestedUsage))

			if allowed, exceeded := quota.LessThanOrEqual(maskedNewUsage, resourceQuota.Hard); !allowed {
				failedRequestedUsage := quota.Mask(requestedUsage, exceeded)
				failedUsed := quota.Mask(resourceQuota.Used, exceeded)
				failedHard := quota.Mask(resourceQuota.Hard, exceeded)
				msg := fmt.Sprintf("resource quota insufficient, requested: %v, used: %v, limited: %v",
					failedRequestedUsage,
					failedUsed,
					failedHard,
				)
				klog.V(3).Infof("enqueueable false for job: %s/%s, because :%s", job.Namespace, job.Name, msg)
				ssn.RecordPodGroupEvent(job.PodGroup, v1.EventTypeNormal, string(scheduling.PodGroupUnschedulableType), msg)
				return util.Reject
			}
		}
		return util.Permit
	})
}

func (rq *resourceQuotaPlugin) OnSessionClose(session *framework.Session) {
}
