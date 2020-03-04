/*
Copyright 2019 The Vulcan Authors.

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

package vresume

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"volcano.sh/volcano/pkg/apis/bus/v1alpha1"
	"volcano.sh/volcano/pkg/cli/util"
)

type resumeFlags struct {
	util.CommonFlags

	Namespace string
	JobName   string
}

var resumeJobFlags = &resumeFlags{}

const (
	// DefaultJobNamespaceEnv is the env name of default namespace of the job
	DefaultJobNamespaceEnv = "VOLCANO_DEFAULT_JOB_NAMESPACE"

	defaultJobNamespace = "default"
)

// InitResumeFlags   init resume command flags
func InitResumeFlags(cmd *cobra.Command) {
	util.InitFlags(cmd, &resumeJobFlags.CommonFlags)

	cmd.Flags().StringVarP(&resumeJobFlags.Namespace, "namespace", "N", "",
		fmt.Sprintf("the namespace of job, overwrite the value of '%s' (default \"%s\")", DefaultJobNamespaceEnv, defaultJobNamespace))
	cmd.Flags().StringVarP(&resumeJobFlags.JobName, "name", "n", "", "the name of job")

	setDefaultArgs()
}

func setDefaultArgs() {

	if resumeJobFlags.Namespace == "" {
		namespace := os.Getenv(DefaultJobNamespaceEnv)

		if namespace != "" {
			resumeJobFlags.Namespace = namespace
		} else {
			resumeJobFlags.Namespace = defaultJobNamespace
		}
	}

}

// ResumeJob  resumes the job
func ResumeJob() error {
	config, err := util.BuildConfig(resumeJobFlags.Master, resumeJobFlags.Kubeconfig)
	if err != nil {
		return err
	}
	if resumeJobFlags.JobName == "" {
		err := fmt.Errorf("job name is mandatory to resume a particular job")
		return err
	}

	return util.CreateJobCommand(config,
		resumeJobFlags.Namespace, resumeJobFlags.JobName,
		v1alpha1.ResumeJobAction)
}
