/*
Copyright 2025.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AgentSpec defines the desired state of Agent
type AgentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Type is the agent type (e.g. scouting-agent)
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Image is the container image
	// +kubebuilder:validation:Required
	Image string `json:"image"`

	// Env is the optional environment variables
	// +optional
	Env []corev1.EnvVar `json:"env,omitempty"`

	// RunOnce indicates if the agent should run to completion (one-shot) or run continuously.
	// Defaults to false (long-running).
	// +optional
	// +kubebuilder:default:=false
	RunOnce bool `json:"runOnce,omitempty"`

	// MaxRestarts limits the number of times the operator will attempt to restart a failing pod
	// for long-running agents (runOnce=false). Defaults to 5. Set to -1 for infinite restarts.
	// Ignored if runOnce is true.
	// +optional
	// +kubebuilder:default:=5
	// +kubebuilder:validation:Minimum:=-1
	MaxRestarts int `json:"maxRestarts,omitempty"`

	// TTL defines the maximum time (in seconds) that an agent can be inactive before being automatically deleted.
	// A value of 0 (default) means no TTL (agent is not ephemeral).
	// +optional
	// +kubebuilder:default:=0
	TTL int64 `json:"ttl,omitempty"`

	// LastActivityTime is the last time the agent was actively used (e.g., received a message or executed code).
	// Updated automatically by the controller or agent runtime.
	// +optional
	LastActivityTime *metav1.Time `json:"lastActivityTime,omitempty"`

	// InputSchemaRef is (future) Input schema
	// +optional
	InputSchemaRef string `json:"inputSchemaRef,omitempty"`

	// OutputSchemaRef is (future) Output schema
	// +optional
	OutputSchemaRef string `json:"outputSchemaRef,omitempty"`
}

// AgentStatus defines the observed state of Agent
type AgentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase is the agent phase (e.g. Pending, Running, Completed, Failed)
	// +optional
	Phase string `json:"phase,omitempty"`

	// Message is the status description
	// +optional
	Message string `json:"message,omitempty"`

	// RestartCount tracks the number of times the pod has been restarted by the operator.
	// Only relevant for long-running agents (runOnce=false).
	// +optional
	RestartCount int `json:"restartCount,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
//+kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Agent is the Schema for the agents API
type Agent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentSpec   `json:"spec,omitempty"`
	Status AgentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AgentList contains a list of Agent
type AgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Agent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Agent{}, &AgentList{})
}
