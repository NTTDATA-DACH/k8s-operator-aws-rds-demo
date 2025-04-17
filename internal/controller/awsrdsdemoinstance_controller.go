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
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	awsRds "github.com/aws/aws-sdk-go-v2/service/rds"
	awsRdsTypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	awsv1alpha1 "github.com/NTTDATA-DACH/k8s-operator-aws-rds-demo/api/v1alpha1"
)

const awsrdsdemoinstanceFinalizer = "aws.nttdata.com/finalizer"

// AwsRDSDemoInstanceReconciler reconciles a AwsRDSDemoInstance object
type AwsRDSDemoInstanceReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	rdsClient *awsRds.Client
}

// +kubebuilder:rbac:groups=aws.nttdata.com,resources=awsrdsdemoinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=aws.nttdata.com,resources=awsrdsdemoinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=aws.nttdata.com,resources=awsrdsdemoinstances/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the AwsRDSDemoInstance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
func (r *AwsRDSDemoInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info(fmt.Sprintf("reconcile triggered for '%s' in namespace '%s'...", req.Name, req.Namespace))

	// Fetch the AwsRDSDemoInstance instance
	instance := &awsv1alpha1.AwsRDSDemoInstance{}
	if err := r.Get(ctx, req.NamespacedName, instance); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("not found AwsRDSDemoInstance, exiting reconciliation loop...")
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
	r.rdsClient = awsRds.NewFromConfig(cfg)

	dbIdentifier := fmt.Sprintf("%s-%s", instance.Spec.DBName, instance.Spec.Stage)

	// Add finalizer
	if !controllerutil.ContainsFinalizer(instance, awsrdsdemoinstanceFinalizer) {
		log.Info("adding finalizer for AwsRDSDemoInstance")
		if ok := controllerutil.AddFinalizer(instance, awsrdsdemoinstanceFinalizer); !ok {
			err = fmt.Errorf("failed to add finalizer into AwsRDSDemoInstance")
			log.Error(err, "failed to add finalizer into AwsRDSDemoInstance")
			return ctrl.Result{}, err
		}
		if err := r.Update(ctx, instance); err != nil {
			log.Error(err, "failed to update AwsRDSDemoInstance to add finalizer: "+err.Error())
			return ctrl.Result{}, err
		}
		log.Info("added finalizer for AwsRDSDemoInstance")
	}

	// Remove instance with finalizer
	isInstanceToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isInstanceToBeDeleted {
		if controllerutil.ContainsFinalizer(instance, awsrdsdemoinstanceFinalizer) {
			log.Info("starting to delete database: " + dbIdentifier)
			if err := r.deleteRDSInstance(ctx, dbIdentifier); err != nil {
				log.Error(err, "failed to remove database: "+err.Error())
				return ctrl.Result{}, err
			}
			log.Info("database deleted, removing finalizer")
			if ok := controllerutil.RemoveFinalizer(instance, awsrdsdemoinstanceFinalizer); !ok {
				err = fmt.Errorf("failed to remove finalizer from AwsRDSDemoInstance")
				log.Error(err, "failed to remove finalizer from AwsRDSDemoInstance")
				return ctrl.Result{}, err
			}
			if err := r.Update(ctx, instance); err != nil {
				log.Error(err, "failed to update AwsRDSDemoInstance to remove finalizer: "+err.Error())
				return ctrl.Result{}, err
			}
		}
	}

	// Fetch credentials from existing secret
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: instance.Spec.CredentialSecretName, Namespace: req.Namespace}, secret); err != nil {
		log.Error(err, "unable to fetch credentials secret"+err.Error())
		return ctrl.Result{}, err
	}
	unb, uexists := secret.Data["username"]
	psb, pexists := secret.Data["password"]
	if !uexists || !pexists {
		err := fmt.Errorf("secret must contain 'username' and 'password'")
		log.Error(err, "invalid secret: "+err.Error())
		return ctrl.Result{}, err
	}
	un := string(unb)
	ps := string(psb)

	// Create db according to the specification
	if err = r.createRDSInstance(ctx, dbIdentifier, instance, un, ps); err != nil {
		log.Error(err, "failed to create db instance: "+err.Error())
		return ctrl.Result{}, err
	}

	// Update db instance if spec changed
	if err = r.updateRDSInstance(ctx, dbIdentifier, instance); err != nil {
		log.Error(err, "failed to update db instance: "+err.Error())
		return ctrl.Result{}, err
	}

	log.Info("reconcile finished...")
	return ctrl.Result{}, nil
}

func (r *AwsRDSDemoInstanceReconciler) createRDSInstance(ctx context.Context, dbIdentifier string, instance *awsv1alpha1.AwsRDSDemoInstance, username string, password string) error {
	log := log.FromContext(ctx)

	_, err := r.rdsClient.DescribeDBInstances(ctx, &awsRds.DescribeDBInstancesInput{
		DBInstanceIdentifier: &dbIdentifier,
	})
	if err != nil {
		log.Info("starting to create db instance: " + dbIdentifier)
		_, err = r.rdsClient.CreateDBInstance(ctx, &awsRds.CreateDBInstanceInput{
			DBInstanceIdentifier: aws.String(dbIdentifier),
			DBInstanceClass:      aws.String(instance.Spec.DBInstanceClass),
			Engine:               aws.String(instance.Spec.Engine),
			EngineVersion:        aws.String(instance.Spec.EngineVersion),
			AllocatedStorage:     aws.Int32(20),
			DBName:               aws.String(instance.Spec.DBName),
			MasterUsername:       aws.String(username),
			MasterUserPassword:   aws.String(password),
		})
		if err != nil {
			return err
		}
		log.Info(fmt.Sprintf("db instance for engine '%s:%s' with the name '%s' with identifier '%s' created successfully", instance.Spec.Engine, instance.Spec.EngineVersion, instance.Spec.DBName, dbIdentifier))
	}

	return nil
}

func (r *AwsRDSDemoInstanceReconciler) updateRDSInstance(ctx context.Context, dbIdentifier string, instance *awsv1alpha1.AwsRDSDemoInstance) error {
	log := log.FromContext(ctx)

	modifyInput := &awsRds.ModifyDBInstanceInput{
		DBInstanceIdentifier: aws.String(dbIdentifier),
		ApplyImmediately:     aws.Bool(true),
	}

	currentDb, err := r.getRDSInstance(ctx, dbIdentifier)
	if err != nil {
		return err
	}

	isInstanceToBeUpdate := false
	if aws.ToString(currentDb.DBInstanceClass) != instance.Spec.DBInstanceClass {
		modifyInput.DBInstanceClass = aws.String(instance.Spec.DBInstanceClass)
		isInstanceToBeUpdate = true
	}
	if aws.ToString(currentDb.EngineVersion) != instance.Spec.EngineVersion {
		modifyInput.EngineVersion = aws.String(instance.Spec.EngineVersion)
		isInstanceToBeUpdate = true
	}

	if isInstanceToBeUpdate {
		log.Info("starting to update db instance: " + dbIdentifier)
		if _, err = r.rdsClient.ModifyDBInstance(ctx, modifyInput); err != nil {
			return err
		}
		log.Info("db instance updated successfully")
	}

	return nil
}

func (r *AwsRDSDemoInstanceReconciler) deleteRDSInstance(ctx context.Context, dbIdentifier string) error {
	log := log.FromContext(ctx)
	log.Info("starting to delete db instance: " + dbIdentifier)

	_, err := r.rdsClient.DeleteDBInstance(ctx, &awsRds.DeleteDBInstanceInput{
		DBInstanceIdentifier: aws.String(dbIdentifier),
		SkipFinalSnapshot:    aws.Bool(true),
	})
	if err != nil {
		var alreadyDeleted *awsRdsTypes.DBInstanceNotFoundFault
		if errors.As(err, &alreadyDeleted) {
			return nil
		}
		return err
	}

	log.Info("db instance delete successfully")
	return nil
}

func (r *AwsRDSDemoInstanceReconciler) getRDSInstance(ctx context.Context, dbIdentifier string) (*awsRdsTypes.DBInstance, error) {
	dbs, err := r.rdsClient.DescribeDBInstances(ctx, &awsRds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(dbIdentifier),
	})
	if err != nil {
		var instanceNotFound *awsRdsTypes.DBInstanceNotFoundFault
		if errors.As(err, &instanceNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if len(dbs.DBInstances) == 0 {
		return nil, nil
	}
	return &dbs.DBInstances[0], nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AwsRDSDemoInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&awsv1alpha1.AwsRDSDemoInstance{}).
		Complete(r)
}
