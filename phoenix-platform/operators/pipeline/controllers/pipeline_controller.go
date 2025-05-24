package controllers

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	phoenixv1alpha1 "github.com/phoenix/platform/operators/pipeline/api/v1alpha1"
)

const (
	finalizerName = "phoenix.io/pipeline-finalizer"
)

// PipelineReconciler reconciles a PhoenixProcessPipeline object
type PipelineReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=phoenix.io,resources=phoenixprocesspipelines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=phoenix.io,resources=phoenixprocesspipelines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=phoenix.io,resources=phoenixprocesspipelines/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *PipelineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the PhoenixProcessPipeline instance
	pipeline := &phoenixv1alpha1.PhoenixProcessPipeline{}
	if err := r.Get(ctx, req.NamespacedName, pipeline); err != nil {
		if errors.IsNotFound(err) {
			// Object not found, could have been deleted
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch PhoenixProcessPipeline")
		return ctrl.Result{}, err
	}

	// Check if the pipeline is being deleted
	if !pipeline.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, pipeline)
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(pipeline, finalizerName) {
		controllerutil.AddFinalizer(pipeline, finalizerName)
		if err := r.Update(ctx, pipeline); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Verify ConfigMap exists
	configMap := &corev1.ConfigMap{}
	configMapKey := types.NamespacedName{
		Namespace: pipeline.Namespace,
		Name:      pipeline.Spec.ConfigMap,
	}
	if err := r.Get(ctx, configMapKey, configMap); err != nil {
		if errors.IsNotFound(err) {
			log.Error(err, "ConfigMap not found", "configMap", pipeline.Spec.ConfigMap)
			return r.updateStatus(ctx, pipeline, "Failed", "ConfigMap not found")
		}
		return ctrl.Result{}, err
	}

	// Create or update DaemonSet
	daemonSet := r.buildDaemonSet(pipeline)
	if err := controllerutil.SetControllerReference(pipeline, daemonSet, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	found := &appsv1.DaemonSet{}
	err := r.Get(ctx, types.NamespacedName{Name: daemonSet.Name, Namespace: daemonSet.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating DaemonSet", "name", daemonSet.Name)
		if err := r.Create(ctx, daemonSet); err != nil {
			log.Error(err, "Failed to create DaemonSet")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// Update DaemonSet if needed
	if !r.daemonSetEqual(found, daemonSet) {
		found.Spec = daemonSet.Spec
		if err := r.Update(ctx, found); err != nil {
			log.Error(err, "Failed to update DaemonSet")
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// Create or update Service for metrics
	service := r.buildService(pipeline)
	if err := controllerutil.SetControllerReference(pipeline, service, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	foundService := &corev1.Service{}
	err = r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Service", "name", service.Name)
		if err := r.Create(ctx, service); err != nil {
			log.Error(err, "Failed to create Service")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// Update status
	return r.updateStatusFromDaemonSet(ctx, pipeline, found)
}

func (r *PipelineReconciler) handleDeletion(ctx context.Context, pipeline *phoenixv1alpha1.PhoenixProcessPipeline) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(pipeline, finalizerName) {
		// Perform cleanup
		log.Info("Performing cleanup for pipeline", "name", pipeline.Name)

		// Remove finalizer
		controllerutil.RemoveFinalizer(pipeline, finalizerName)
		if err := r.Update(ctx, pipeline); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *PipelineReconciler) buildDaemonSet(pipeline *phoenixv1alpha1.PhoenixProcessPipeline) *appsv1.DaemonSet {
	labels := map[string]string{
		"app":                      "phoenix-collector",
		"phoenix.io/pipeline":      pipeline.Name,
		"phoenix.io/experiment-id": pipeline.Spec.ExperimentID,
		"phoenix.io/variant":       pipeline.Spec.Variant,
		"prometheus.io/scrape":     "true",
	}

	hostPathDirectory := corev1.HostPathDirectory
	
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipeline.Name,
			Namespace: pipeline.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						"prometheus.io/port": "8888",
						"prometheus.io/path": "/metrics",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: pipeline.Spec.ServiceAccount,
					HostNetwork:        true,
					HostPID:            pipeline.Spec.RequiresHostPID,
					Containers: []corev1.Container{
						{
							Name:  "otel-collector",
							Image: pipeline.Spec.CollectorImage,
							Args: []string{
								"--config=/etc/otel/config.yaml",
							},
							Env: []corev1.EnvVar{
								{
									Name: "NEW_RELIC_API_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "newrelic-secret",
											},
											Key: "api-key",
										},
									},
								},
								{
									Name:  "PHOENIX_EXPERIMENT_ID",
									Value: pipeline.Spec.ExperimentID,
								},
								{
									Name:  "PHOENIX_VARIANT",
									Value: pipeline.Spec.Variant,
								},
								{
									Name: "NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
							},
							Resources: pipeline.Spec.Resources,
							Ports: []corev1.ContainerPort{
								{
									Name:          "metrics",
									ContainerPort: 8888,
									Protocol:      corev1.ProtocolTCP,
								},
								{
									Name:          "health",
									ContainerPort: 13133,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "config",
									MountPath: "/etc/otel",
									ReadOnly:  true,
								},
								{
									Name:      "hostfs",
									MountPath: "/hostfs",
									ReadOnly:  true,
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(13133),
									},
								},
								InitialDelaySeconds: 15,
								PeriodSeconds:       10,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(13133),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: pipeline.Spec.ConfigMap,
									},
								},
							},
						},
						{
							Name: "hostfs",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/",
									Type: &hostPathDirectory,
								},
							},
						},
					},
					NodeSelector: pipeline.Spec.NodeSelector,
					Tolerations:  pipeline.Spec.Tolerations,
				},
			},
		},
	}

	// Set default resources if not specified
	if ds.Spec.Template.Spec.Containers[0].Resources.Requests == nil {
		ds.Spec.Template.Spec.Containers[0].Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("100m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		}
	}
	if ds.Spec.Template.Spec.Containers[0].Resources.Limits == nil {
		ds.Spec.Template.Spec.Containers[0].Resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("500m"),
			corev1.ResourceMemory: resource.MustParse("512Mi"),
		}
	}

	return ds
}

func (r *PipelineReconciler) buildService(pipeline *phoenixv1alpha1.PhoenixProcessPipeline) *corev1.Service {
	labels := map[string]string{
		"app":                      "phoenix-collector",
		"phoenix.io/pipeline":      pipeline.Name,
		"phoenix.io/experiment-id": pipeline.Spec.ExperimentID,
		"phoenix.io/variant":       pipeline.Spec.Variant,
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-metrics", pipeline.Name),
			Namespace: pipeline.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "metrics",
					Port:       8888,
					TargetPort: intstr.FromString("metrics"),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

func (r *PipelineReconciler) daemonSetEqual(a, b *appsv1.DaemonSet) bool {
	// Compare relevant fields
	if a.Spec.Template.Spec.Containers[0].Image != b.Spec.Template.Spec.Containers[0].Image {
		return false
	}
	
	// Compare ConfigMap name
	if a.Spec.Template.Spec.Volumes[0].ConfigMap.Name != b.Spec.Template.Spec.Volumes[0].ConfigMap.Name {
		return false
	}

	return true
}

func (r *PipelineReconciler) updateStatus(ctx context.Context, pipeline *phoenixv1alpha1.PhoenixProcessPipeline, phase string, message string) (ctrl.Result, error) {
	pipeline.Status.Phase = phase
	pipeline.Status.LastUpdated = metav1.NewTime(time.Now())
	pipeline.Status.ObservedGeneration = pipeline.Generation

	// Update condition
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		Reason:             phase,
		Message:            message,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}
	if phase == "Running" {
		condition.Status = metav1.ConditionTrue
	}

	// Update or append condition
	found := false
	for i, c := range pipeline.Status.Conditions {
		if c.Type == condition.Type {
			pipeline.Status.Conditions[i] = condition
			found = true
			break
		}
	}
	if !found {
		pipeline.Status.Conditions = append(pipeline.Status.Conditions, condition)
	}

	if err := r.Status().Update(ctx, pipeline); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

func (r *PipelineReconciler) updateStatusFromDaemonSet(ctx context.Context, pipeline *phoenixv1alpha1.PhoenixProcessPipeline, ds *appsv1.DaemonSet) (ctrl.Result, error) {
	pipeline.Status.ReadyNodes = ds.Status.NumberReady
	pipeline.Status.TotalNodes = ds.Status.DesiredNumberScheduled

	phase := "Running"
	message := fmt.Sprintf("%d/%d nodes ready", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
	
	if ds.Status.NumberReady == 0 {
		phase = "Pending"
		message = "Waiting for pods to be ready"
	} else if ds.Status.NumberReady < ds.Status.DesiredNumberScheduled {
		phase = "Running"
		message = fmt.Sprintf("Partial deployment: %s", message)
	}

	return r.updateStatus(ctx, pipeline, phase, message)
}

// SetupWithManager sets up the controller with the Manager.
func (r *PipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&phoenixv1alpha1.PhoenixProcessPipeline{}).
		Owns(&appsv1.DaemonSet{}).
		Owns(&corev1.Service{}).
		Complete(r)
}