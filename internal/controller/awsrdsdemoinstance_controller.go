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
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	awsRds "github.com/aws/aws-sdk-go-v2/service/rds"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	//	awsRdsTypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	//	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	//	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	awsv1alpha1 "github.com/NTTDATA-DACH/k8s-operator-aws-rds-demo/api/v1alpha1"
)

const awsrdsdemoinstanceFinalizer = "aws.nttdata.com/finalizer"

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
// the AwsRDSDemoInstance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
func (r *AwsRDSDemoInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("reconcile triggered...")

	// Fetch the AwsRDSDemoInstance instance
	instance := &awsv1alpha1.AwsRDSDemoInstance{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if apierrors.IsNotFound(err) {
			log.Error(err, "not found: "+err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	log.Info("found: " + instance.Name)

	// Create RDS client
	cfg, err := awsCfg.LoadDefaultConfig(ctx)
	if err != nil {
		log.Error(err, "failed to load AWS config: "+err.Error())
		return ctrl.Result{}, err
	}
	rdsClient := awsRds.NewFromConfig(cfg)

	dbIdentifier := fmt.Sprintf("%s-%s", instance.Spec.DBName, instance.Spec.Stage)

	// Add finalizer
	if !controllerutil.ContainsFinalizer(instance, awsrdsdemoinstanceFinalizer) {
		log.Info("adding finalizer for AwsRDSDemoInstance")
		if ok := controllerutil.AddFinalizer(instance, awsrdsdemoinstanceFinalizer); !ok {
			err = fmt.Errorf("failed to add finalizer into AwsRDSDemoInstance")
			log.Error(err, "failed to add finalizer into AwsRDSDemoInstance")
			return ctrl.Result{}, err
		}
		if err = r.Update(ctx, instance); err != nil {
			log.Error(err, "failed to update AwsRDSDemoInstance to add finalizer: "+err.Error())
			return ctrl.Result{}, err
		}
		log.Info("added finalizer for AwsRDSDemoInstance")
	}

	// Fetch credentials from existing secret
	/*
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: instance.Spec.CredentialSecretName, Namespace: req.Namespace}, secret); err != nil {
			log.Error(err, "unable to fetch credentials secret"+err.Error())
			return ctrl.Result{}, err
		}
		unb, uexists := secret.Data["username"]
		psb, pexists := secret.Data["password"]
		if !uexists || !pexists {
			err := fmt.Errorf("secret must contain 'username' and 'password'")
			log.Error(err, "invalid secret")
			return ctrl.Result{}, err
		}
		un := string(unb)
		ps := string(psb)
	*/
	un, ps := "postgres", "postgres"

	// Create db according to the specification
	_, err = rdsClient.DescribeDBInstances(ctx, &awsRds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &dbIdentifier,
	})
	if err != nil {
		_, err = rdsClient.CreateDBInstance(ctx, &awsRds.CreateDBInstanceInput{
			DBInstanceIdentifier: aws.String(dbIdentifier),
			DBInstanceClass:      aws.String(instance.Spec.DBInstanceClass),
			Engine:               aws.String(instance.Spec.Engine),
			EngineVersion:        aws.String(instance.Spec.EngineVersion),
			AllocatedStorage:     aws.Int32(20),
			DBName:               aws.String(instance.Spec.DBName),
			MasterUsername:       aws.String(un),
			MasterUserPassword:   aws.String(ps),
		})
		if err != nil {
			log.Error(err, "failed to create db: "+err.Error())
			return ctrl.Result{}, err
		}
		log.Info(fmt.Sprintf("database engine '%s:%s' with the name '%s' with identifier '%s' created successfully", instance.Spec.Engine, instance.Spec.EngineVersion, instance.Spec.DBName, dbIdentifier))
	}

	log.Info("reconcile finished...")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AwsRDSDemoInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&awsv1alpha1.AwsRDSDemoInstance{}).
		Complete(r)
}
