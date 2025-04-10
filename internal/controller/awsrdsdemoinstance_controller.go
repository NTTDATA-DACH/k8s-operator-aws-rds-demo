/*
Copyright 2025 NTT DATA.

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

package controller

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	awsv1alpha1 "github.com/NTTDATA-DACH/k8s-operator-aws-rds-demo/api/v1alpha1"
)

// AwsRDSDemoInstanceReconciler reconciles a AwsRDSDemoInstance object
type AwsRDSDemoInstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=aws.nttdata.com,resources=awsrdsdemoinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws.nttdata.com,resources=awsrdsdemoinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws.nttdata.com,resources=awsrdsdemoinstances/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AwsRDSDemoInstance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *AwsRDSDemoInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("reconcile triggered")

	instance := &awsv1alpha1.AwsRDSDemoInstance{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Create AWS session and RDS client
	sess := session.Must(session.NewSession())
	svc := rds.New(sess)

	// Check if RDS already exists
	di := &rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instance.Spec.InstanceID),
	}
	if _, err := svc.DescribeDBInstances(di); err == nil {
		logger.Info("db already exists")
		instance.Status.Status = "AlreadyExists"
		r.Status().Update(ctx, instance)
		return ctrl.Result{}, nil
	}

	// RDS does not exist, try to create one
	logger.Info("db does not exits, try to create one")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AwsRDSDemoInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&awsv1alpha1.AwsRDSDemoInstance{}).
		Complete(r)
}
