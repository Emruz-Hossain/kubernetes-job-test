package cmd

import (
	batch "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core "k8s.io/api/core/v1"
)

func NewJob(name string) batch.Job{
	return batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: batch.JobSpec{
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: "busybox-pod",
				},
				Spec: core.PodSpec{
					Containers:[]core.Container{
						{
							Name: "busybox",
							Image:"busybox",
							Command:[]string{
								"sleep",
								"120",
							},
						},
					},
					RestartPolicy:core.RestartPolicyOnFailure,
				},

			},
		},
	}
}