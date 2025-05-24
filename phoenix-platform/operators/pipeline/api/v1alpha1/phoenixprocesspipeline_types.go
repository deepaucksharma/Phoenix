package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PhoenixProcessPipelineSpec defines the desired state of PhoenixProcessPipeline
type PhoenixProcessPipelineSpec struct {
	// ExperimentID is the ID of the experiment this pipeline belongs to
	ExperimentID string `json:"experimentID"`

	// Variant is either "baseline" or "candidate"
	Variant string `json:"variant"`

	// ConfigMap is the name of the ConfigMap containing the OTel collector configuration
	ConfigMap string `json:"configMap"`

	// CollectorImage is the OTel collector image to use
	// +kubebuilder:default="otel/opentelemetry-collector-contrib:0.88.0"
	CollectorImage string `json:"collectorImage,omitempty"`

	// NodeSelector for pod assignment
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations for pod assignment
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// RequiresHostPID indicates if the collector needs host PID namespace
	// +kubebuilder:default=false
	RequiresHostPID bool `json:"requiresHostPID,omitempty"`

	// Resources for the collector pods
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// ServiceAccount to use for the collector pods
	// +kubebuilder:default="phoenix-collector"
	ServiceAccount string `json:"serviceAccount,omitempty"`
}

// PhoenixProcessPipelineStatus defines the observed state of PhoenixProcessPipeline
type PhoenixProcessPipelineStatus struct {
	// Phase represents the current phase of the pipeline
	// +kubebuilder:validation:Enum=Pending;Running;Failed
	Phase string `json:"phase,omitempty"`

	// ReadyNodes is the number of nodes where the collector is ready
	ReadyNodes int32 `json:"readyNodes,omitempty"`

	// TotalNodes is the total number of nodes where the collector should run
	TotalNodes int32 `json:"totalNodes,omitempty"`

	// Conditions represent the latest available observations of the pipeline's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastUpdated is the last time the status was updated
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ppp
// +kubebuilder:printcolumn:name="Experiment",type="string",JSONPath=".spec.experimentID"
// +kubebuilder:printcolumn:name="Variant",type="string",JSONPath=".spec.variant"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.readyNodes"
// +kubebuilder:printcolumn:name="Total",type="string",JSONPath=".status.totalNodes"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// PhoenixProcessPipeline is the Schema for the phoenixprocesspipelines API
type PhoenixProcessPipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PhoenixProcessPipelineSpec   `json:"spec,omitempty"`
	Status PhoenixProcessPipelineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PhoenixProcessPipelineList contains a list of PhoenixProcessPipeline
type PhoenixProcessPipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PhoenixProcessPipeline `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PhoenixProcessPipeline{}, &PhoenixProcessPipelineList{})
}