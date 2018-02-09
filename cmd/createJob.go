// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/appscode/go/homedir"
	"github.com/appscode/kutil"
	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	apps "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// createJobCmd represents the createJob command
var createJobCmd = &cobra.Command{
	Use:   "createJob",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		configpath := getKubeConfigPath()
		config, err := clientcmd.BuildConfigFromFlags("", configpath)
		if err != nil {
			log.Fatalf("Faield to create kube config. Reason:%v", err.Error())
		}

		kubeClient := kubernetes.NewForConfigOrDie(config)

		/*------------------Scale down replica------------------*/
		obj, err := kubeClient.AppsV1().Deployments("default").Get("stash-demo", metav1.GetOptions{})

		_, _, err = PatchDeployment(kubeClient, obj, func(dp *apps.Deployment) *apps.Deployment {
			var replica int32
			replica = 0
			dp.Spec.Replicas = &replica
			return dp
		})
		/*--------------------Creating Job----------------------------*/
		job := NewJob("test-job")
		jobObj, err := kubeClient.BatchV1().Jobs("default").Create(&job)
		fmt.Println("Error: ", err)
		fmt.Println("Job created. Name:", jobObj.Name)
		WaitUntilJobCompleted(kubeClient, "test-job")
		fmt.Println("job completed")
		/*------------------------Deleting Job---------------------*/
		kubeClient.BatchV1().Jobs("default").Delete("test-job", deleteInBackground())
		fmt.Println("job deleted")
		/*----------------------Scale up replica-------------------*/
		obj, err = kubeClient.AppsV1().Deployments("default").Get("stash-demo", metav1.GetOptions{})

		_, _, err = PatchDeployment(kubeClient, obj, func(dp *apps.Deployment) *apps.Deployment {
			var replica int32
			replica = 5
			dp.Spec.Replicas = &replica
			return dp
		})
	},
}

func init() {
	RootCmd.AddCommand(createJobCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// createJobCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// createJobCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func getKubeConfigPath() string {

	var kubeConfigPath string

	homeDir := homedir.HomeDir()

	if _, err := os.Stat(homeDir + "/.kube/config"); err == nil {
		kubeConfigPath = homeDir + "/.kube/config"
	} else {
		fmt.Printf("Enter kubernetes config directory: ")
		fmt.Scanf("%s", kubeConfigPath)
	}

	return kubeConfigPath
}

func WaitUntilJobCompleted(kubeClient *kubernetes.Clientset, name string) error {
	return backoff.Retry(func() error {
		job, err := kubeClient.BatchV1().Jobs("default").Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		fmt.Println(job.Status.Succeeded)
		if job.Status.Succeeded == 1 {
			return nil
		}
		fmt.Println("Retrying....")
		return errors.New("check again")
	}, backoff.NewConstantBackOff(3*time.Second))
}
func deleteInBackground() *metav1.DeleteOptions {
	policy := metav1.DeletePropagationBackground
	return &metav1.DeleteOptions{PropagationPolicy: &policy}
}
func PatchDeployment(c *kubernetes.Clientset, cur *apps.Deployment, transform func(*apps.Deployment) *apps.Deployment) (*apps.Deployment, kutil.VerbType, error) {
	curJson, err := json.Marshal(cur)
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	modJson, err := json.Marshal(transform(cur.DeepCopy()))
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(curJson, modJson, apps.Deployment{})
	if err != nil {
		return nil, kutil.VerbUnchanged, err
	}
	if len(patch) == 0 || string(patch) == "{}" {
		return cur, kutil.VerbUnchanged, nil
	}
	glog.V(3).Infof("Patching Deployment %s/%s with %s.", cur.Namespace, cur.Name, string(patch))
	out, err := c.AppsV1().Deployments(cur.Namespace).Patch(cur.Name, types.StrategicMergePatchType, patch)

	return out, kutil.VerbPatched, err
}
