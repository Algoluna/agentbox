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

package controller

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors" // Alias to avoid confusion with standard errors pkg
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/lib/pq"
	_ "github.com/lib/pq" // Import postgres driver

	"github.com/redis/go-redis/v9"

	agentsv1alpha1 "github.com/Algoluna/agent-operator/api/v1alpha1"
)

const (
	postgresSecretMountPath = "/etc/secrets/postgres"
	valkeySecretMountPath   = "/etc/secrets/valkey" // Define even if not used yet
)

// AgentReconciler reconciles a Agent object
type AgentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	PhasePending   = "Pending"
	PhaseRunning   = "Running"
	PhaseCompleted = "Completed"
	PhaseFailed    = "Failed"
)

// +kubebuilder:rbac:groups=agents.algoluna.com,resources=agents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agents.algoluna.com,resources=agents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agents.algoluna.com,resources=agents/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete # Added Secret permissions

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *AgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	var err error

	// Fetch the Agent instance
	var agent agentsv1alpha1.Agent
	if err := r.Get(ctx, req.NamespacedName, &agent); err != nil {
		if errors.IsNotFound(err) {
			// Agent was deleted - no need to requeue
			log.Info("Agent resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Agent")
		return ctrl.Result{}, err
	}

	// --- Ensure dedicated namespace for this agent type ---
	agentTypeNamespace := fmt.Sprintf("agent-%s", agent.Spec.Type)
	var ns corev1.Namespace
	err = r.Get(ctx, types.NamespacedName{Name: agentTypeNamespace}, &ns)
	if err != nil && apierrors.IsNotFound(err) {
		// Namespace does not exist, create it
		ns = corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: agentTypeNamespace,
			},
		}
		if createErr := r.Create(ctx, &ns); createErr != nil && !apierrors.IsAlreadyExists(createErr) {
			log.Error(createErr, "Failed to create agent type namespace", "Namespace", agentTypeNamespace)
			return ctrl.Result{}, createErr
		}
		log.Info("Created dedicated namespace for agent type", "Namespace", agentTypeNamespace)
	} else if err != nil {
		log.Error(err, "Failed to get agent type namespace", "Namespace", agentTypeNamespace)
		return ctrl.Result{}, err
	}
	// If the Agent CR is not in the correct namespace, move it (not supported directly, so log a warning)
	if agent.Namespace != agentTypeNamespace {
		log.Info("Agent CR is not in the correct type-based namespace. Please create Agent CRs in the namespace: " + agentTypeNamespace)
		return ctrl.Result{}, nil
	}

	// --- Start of Secret Provisioning Logic ---
	postgresSecretName := fmt.Sprintf("agent-%s-postgres-creds", agent.Spec.Type)
	valkeySecretName := fmt.Sprintf("agent-%s-valkey-creds", agent.Spec.Type) // Define even if not used yet

	// Check if Postgres secret exists
	var pgSecret corev1.Secret
	err = r.Get(ctx, types.NamespacedName{Name: postgresSecretName, Namespace: agent.Namespace}, &pgSecret)
	if err != nil && apierrors.IsNotFound(err) {
		log.Info("Postgres secret not found, attempting to provision", "SecretName", postgresSecretName, "Namespace", agent.Namespace)
		// Secret doesn't exist, provision it
		createdSecretName, provisionErr := r.provisionPostgresCredentials(ctx, &agent)
		if provisionErr != nil {
			log.Error(provisionErr, "Failed to provision Postgres credentials and secret")
			// Update status and requeue with backoff
			_, statusErr := r.updateAgentStatus(ctx, &agent, PhaseFailed, fmt.Sprintf("Failed to provision credentials: %v", provisionErr))
			return ctrl.Result{RequeueAfter: time.Second * 30}, statusErr // Requeue after delay
		}
		postgresSecretName = createdSecretName // Use the name returned (should be the same)
		log.Info("Successfully provisioned Postgres secret", "SecretName", postgresSecretName)
		// Requeue immediately to proceed with pod creation now that secret exists
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		// Other error getting secret
		log.Error(err, "Failed to get Postgres secret", "SecretName", postgresSecretName)
		return ctrl.Result{}, err
	}
	// --- Valkey Secret Provisioning ---
	var valkeySecret corev1.Secret
	err = r.Get(ctx, types.NamespacedName{Name: valkeySecretName, Namespace: agent.Namespace}, &valkeySecret)
	if err != nil && apierrors.IsNotFound(err) {
		log.Info("Valkey secret not found, attempting to provision", "SecretName", valkeySecretName, "Namespace", agent.Namespace)
		createdValkeySecretName, provisionErr := r.provisionValkeyCredentials(ctx, &agent)
		if provisionErr != nil {
			log.Error(provisionErr, "Failed to provision Valkey credentials and secret")
			_, statusErr := r.updateAgentStatus(ctx, &agent, PhaseFailed, fmt.Sprintf("Failed to provision valkey credentials: %v", provisionErr))
			return ctrl.Result{RequeueAfter: time.Second * 30}, statusErr
		}
		valkeySecretName = createdValkeySecretName
		log.Info("Successfully provisioned Valkey secret", "SecretName", valkeySecretName)
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Valkey secret", "SecretName", valkeySecretName)
		return ctrl.Result{}, err
	}
	// --- End of Secret Provisioning Logic ---

	// Check if pod already exists for this agent
	podName := fmt.Sprintf("agent-%s", agent.Name)
	var pod corev1.Pod
	podFound := true
	err = r.Get(ctx, types.NamespacedName{Name: podName, Namespace: agent.Namespace}, &pod)

	if err != nil {
		if apierrors.IsNotFound(err) {
			podFound = false
			// Pod doesn't exist, create it if the Agent is not in a terminal state
			if agent.Status.Phase != PhaseCompleted && agent.Status.Phase != PhaseFailed {
				log.Info("Pod not found, creating a new one")
				// Pass the determined secret names to the pod constructor
				newPod := r.constructPodForAgent(&agent, postgresSecretName, valkeySecretName)
				if err := r.Create(ctx, newPod); err != nil {
					log.Error(err, "Failed to create Pod for Agent", "Pod.Namespace", newPod.Namespace, "Pod.Name", newPod.Name)
					// Use apierrors here
					return r.updateAgentStatus(ctx, &agent, PhaseFailed, fmt.Sprintf("Failed to create pod: %v", err))
				}
				log.Info("Created Pod for Agent", "Pod.Namespace", newPod.Namespace, "Pod.Name", newPod.Name)
				return r.updateAgentStatus(ctx, &agent, PhasePending, "Pod created, waiting for it to start")
			}
			// If Agent is Completed/Failed and pod is gone, do nothing
			return ctrl.Result{}, nil
		}
		// Other error getting pod
		log.Error(err, "Failed to get Pod")
		return ctrl.Result{}, err
	}

	// --- Pod Exists ---

	// Update Agent status based on Pod status
	currentAgentPhase := agent.Status.Phase
	newPhase := currentAgentPhase
	newMessage := agent.Status.Message

	switch pod.Status.Phase {
	case corev1.PodRunning:
		newPhase = PhaseRunning
		newMessage = "Agent pod is running"
	case corev1.PodSucceeded:
		// Only transition to Completed if it's a runOnce agent
		if agent.Spec.RunOnce {
			newPhase = PhaseCompleted
			newMessage = "Agent pod completed successfully"
		} else {
			// For long-running agents, Succeeded means it exited unexpectedly. Treat as failure for restart logic.
			log.Info("Long-running agent pod Succeeded unexpectedly, treating as failure for potential restart", "Pod.Name", pod.Name)
			newPhase = PhaseFailed
			newMessage = "Long-running agent pod completed unexpectedly"
			// Proceed to failure handling below
		}
	case corev1.PodFailed:
		newPhase = PhaseFailed
		newMessage = fmt.Sprintf("Agent pod failed: %s", pod.Status.Reason)
		// Proceed to failure handling below
	case corev1.PodPending:
		newPhase = PhasePending
		newMessage = "Agent pod is pending"
	default: // Includes PodUnknown
		newPhase = PhasePending // Or potentially a different status?
		newMessage = fmt.Sprintf("Agent pod in unknown phase: %s", pod.Status.Phase)
	}

	// Handle Failed state for long-running agents (restart logic)
	if newPhase == PhaseFailed && !agent.Spec.RunOnce && podFound {
		// Check MaxRestarts (-1 means infinite)
		if agent.Spec.MaxRestarts == -1 || agent.Status.RestartCount < agent.Spec.MaxRestarts {
			log.Info("Attempting to restart failed/completed long-running agent pod", "RestartCount", agent.Status.RestartCount, "MaxRestarts", agent.Spec.MaxRestarts)

			// Delete the failed pod
			if err := r.Delete(ctx, &pod); err != nil && !errors.IsNotFound(err) {
				log.Error(err, "Failed to delete failed pod for restart", "Pod.Name", pod.Name)
				return r.updateAgentStatus(ctx, &agent, PhaseFailed, fmt.Sprintf("Failed to delete pod %s for restart: %v", pod.Name, err))
			}

			// Increment restart count and update status
			agent.Status.RestartCount++
			_, updateErr := r.updateAgentStatus(ctx, &agent, PhasePending, fmt.Sprintf("Restarting pod (attempt %d)", agent.Status.RestartCount))

			// Requeue after a backoff period (simple example, could use exponential)
			// Note: The reconcile will trigger again when the pod is deleted,
			// and the 'pod not found' logic will create a new one.
			// Adding a small delay might prevent rapid recreation cycles if creation fails immediately.
			requeueDelay := time.Second * 5 // Simple delay, consider exponential backoff later
			log.Info("Requeuing reconciliation after pod deletion for restart", "delay", requeueDelay)
			return ctrl.Result{RequeueAfter: requeueDelay}, updateErr // Return potential status update error

		} else {
			// Max restarts exceeded
			log.Info("Max restarts exceeded for agent pod", "MaxRestarts", agent.Spec.MaxRestarts)
			newMessage = fmt.Sprintf("Agent pod failed and exceeded max restarts (%d): %s", agent.Spec.MaxRestarts, pod.Status.Reason)
			// Status will be updated below
		}
	}

	// Update status if phase or message changed
	if newPhase != currentAgentPhase || newMessage != agent.Status.Message {
		return r.updateAgentStatus(ctx, &agent, newPhase, newMessage)
	}

	// If nothing changed, no need to requeue
	return ctrl.Result{}, nil
}

// updateAgentStatus updates the status of the Agent resource.
func (r *AgentReconciler) updateAgentStatus(ctx context.Context, agent *agentsv1alpha1.Agent, phase string, message string) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	agent.Status.Phase = phase
	agent.Status.Message = message

	// Use retry loop for status updates to handle potential conflicts
	// Use apierrors here
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Fetch the latest version of Agent before attempting update
		// RetryOnConflict uses exponential backoff to avoid exhausting the apiserver
		currentAgent := &agentsv1alpha1.Agent{} // Create a new instance for the Get call
		getErr := r.Get(ctx, types.NamespacedName{Name: agent.Name, Namespace: agent.Namespace}, currentAgent)
		if getErr != nil {
			// If the agent is gone, we can't update status, just return the error
			if apierrors.IsNotFound(getErr) {
				log.Info("Agent not found during status update, likely deleted.")
				return getErr // Propagate the NotFound error
			}
			log.Error(getErr, "Failed to re-fetch Agent for status update")
			return getErr
		}
		// Apply the changes to the fetched object
		currentAgent.Status.Phase = phase
		currentAgent.Status.Message = message
		currentAgent.Status.RestartCount = agent.Status.RestartCount // Ensure restart count is updated

		// Update the status
		return r.Status().Update(ctx, currentAgent)
	})

	// Handle NotFound error specifically after retry loop
	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil // Agent was deleted, stop reconciliation
	} else if err != nil {
		log.Error(err, "Failed to update Agent status after retries")
		return ctrl.Result{}, err
	}
	log.Info("Updated Agent status", "Phase", phase, "Message", message)
	return ctrl.Result{}, nil
}

// constructPodForAgent creates a pod object for the given Agent, injecting secret volumes
func (r *AgentReconciler) constructPodForAgent(agent *agentsv1alpha1.Agent, postgresSecretName, valkeySecretName string) *corev1.Pod {
	log := logf.Log.WithValues("agent", agent.Name, "namespace", agent.Namespace) // Use logger
	podName := fmt.Sprintf("agent-%s", agent.Name)

	// Determine RestartPolicy based on RunOnce
	restartPolicy := corev1.RestartPolicyNever // Default for runOnce=true
	if !agent.Spec.RunOnce {
		restartPolicy = corev1.RestartPolicyOnFailure // Or Always? OnFailure seems better with operator restarts.
	}

	// Convert agent.Spec.Env to corev1.EnvVar
	var envVars []corev1.EnvVar
	for _, env := range agent.Spec.Env {
		envVars = append(envVars, corev1.EnvVar{
			Name:  env.Name,
			Value: env.Value,
		})
	}

	// Add AGENT_ID environment variable
	envVars = append(envVars, corev1.EnvVar{
		Name:  "AGENT_ID",
		Value: agent.Name, // Agent instance name
	})
	envVars = append(envVars, corev1.EnvVar{
		Name:  "AGENT_TYPE", // Agent type
		Value: agent.Spec.Type,
	})

	// Define Volumes based on provided secret names
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	if postgresSecretName != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "postgres-creds",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: postgresSecretName,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "postgres-creds",
			MountPath: postgresSecretMountPath,
			ReadOnly:  true,
		})
		log.Info("Adding postgres secret volume mount", "SecretName", postgresSecretName, "MountPath", postgresSecretMountPath)
	}

	// Add similar logic for valkeySecretName if/when Valkey provisioning is added
	if valkeySecretName != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "valkey-creds",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: valkeySecretName,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "valkey-creds",
			MountPath: valkeySecretMountPath,
			ReadOnly:  true,
		})
		log.Info("Adding valkey secret volume mount", "SecretName", valkeySecretName, "MountPath", valkeySecretMountPath)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: agent.Namespace,
			Labels: map[string]string{
				"app":        "agent",
				"agent-name": agent.Name,
				"agent-type": agent.Spec.Type,
			},
			// Owner reference is set below using SetControllerReference
		},
		Spec: corev1.PodSpec{
			RestartPolicy: restartPolicy, // Set based on RunOnce
			Containers: []corev1.Container{
				{
					Name:            "agent",
					Image:           agent.Spec.Image,
					Env:             envVars,
					ImagePullPolicy: corev1.PullNever, // Keep using local images for now
					VolumeMounts:    volumeMounts,     // Add the volume mounts
				},
			},
			Volumes: volumes, // Add the volumes
		},
	}

	// Set Agent instance as the owner and controller
	if err := controllerutil.SetControllerReference(agent, pod, r.Scheme); err != nil {
		// Log the error, but the reconcile loop should handle it
		log.Error(err, "Failed to set controller reference on pod") // Use instance logger
	}

	return pod
}

/*
provisionValkeyCredentials generates a random password and creates a K8s Secret for Valkey access.
The secret is created in the agent type namespace and is owned by the Agent CR.
*/

func (r *AgentReconciler) provisionValkeyCredentials(ctx context.Context, agent *agentsv1alpha1.Agent) (string, error) {
	log := logf.FromContext(ctx).WithValues("agent", agent.Name, "namespace", agent.Namespace, "agentType", agent.Spec.Type)
	secretName := fmt.Sprintf("agent-%s-valkey-creds", agent.Spec.Type)

	valkeyUser := fmt.Sprintf("agent_%s", createRoleName(agent.Spec.Type))

	// Valkey connection info
	valkeyNamespace := os.Getenv("VALKEY_NAMESPACE")
	if valkeyNamespace == "" {
		valkeyNamespace = "agentbox-system"
	}
	valkeyServiceName := os.Getenv("VALKEY_SERVICE_NAME")
	if valkeyServiceName == "" {
		valkeyServiceName = "agentbox-valkey"
	}
	valkeyFQDN := fmt.Sprintf("%s.%s.svc.cluster.local", valkeyServiceName, valkeyNamespace)
	valkeyPort := os.Getenv("VALKEY_PORT")
	if valkeyPort == "" {
		valkeyPort = "6379"
	}

	// Valkey admin credentials (must be set as env vars or via secret)
	valkeyAdminUser := os.Getenv("VALKEY_ADMIN_USER")
	if valkeyAdminUser == "" {
		valkeyAdminUser = "default"
	}
	valkeyAdminPassword := os.Getenv("VALKEY_ADMIN_PASSWORD")
	if valkeyAdminPassword == "" {
		return "", fmt.Errorf("VALKEY_ADMIN_PASSWORD must be set in the controller environment")
	}

	// Check if the secret already exists
	var valkeySecret corev1.Secret
	secretExists := false
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: agent.Namespace}, &valkeySecret)
	if err == nil {
		secretExists = true
	} else if !apierrors.IsNotFound(err) {
		return "", fmt.Errorf("failed to check for existing valkey secret: %w", err)
	}

	var password string
	if secretExists {
		// Use the password from the existing secret
		pwBytes, ok := valkeySecret.Data["password"]
		if !ok {
			return "", fmt.Errorf("existing valkey secret missing password field")
		}
		password = string(pwBytes)
		log.Info("Using existing Valkey password from secret", "SecretName", secretName)
	} else {
		// Generate a new password
		password, err = generatePassword(32)
		if err != nil {
			return "", fmt.Errorf("failed to generate valkey password: %w", err)
		}
		log.Info("Generated new Valkey password", "SecretName", secretName)
	}

	// Connect to Valkey as admin
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", valkeyFQDN, valkeyPort),
		Username: valkeyAdminUser,
		Password: valkeyAdminPassword,
	})
	defer rdb.Close()

	// Test connection
	ping := rdb.Ping(ctx)
	if ping.Err() != nil {
		return "", fmt.Errorf("failed to connect to Valkey as admin: %w", ping.Err())
	}

	// Set up ACL for agent-type user: full access to <type>:* and system:*
	_, err = rdb.Do(ctx, "ACL", "SETUSER", valkeyUser,
		"on",
		">"+password,
		"~agent:"+agent.Spec.Type+":*",
		"~system:*",
		"+@all",
	).Result()
	if err != nil {
		return "", fmt.Errorf("failed to set ACL for Valkey user %s: %w", valkeyUser, err)
	}
	log.Info("Valkey user created/updated with ACL", "user", valkeyUser, "acl", map[string]interface{}{
		"on":           true,
		"password":     "****",
		"key_patterns": []string{agent.Spec.Type + ":*", "system:*"},
		"commands":     "+@all",
	})

	// Store credentials in secret for agent
	secretData := map[string][]byte{
		"username": []byte(valkeyUser),
		"password": []byte(password),
		"host":     []byte(valkeyFQDN),
		"port":     []byte(valkeyPort),
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: agent.Namespace,
		},
		Data: secretData,
		Type: corev1.SecretTypeOpaque,
	}

	// Set owner reference so Secret is deleted when Agent is deleted
	if err := controllerutil.SetControllerReference(agent, secret, r.Scheme); err != nil {
		log.Error(err, "Failed to set controller reference on valkey secret", "SecretName", secretName)
	}

	if secretExists {
		// Update the secret if needed (e.g., if fields are missing or changed)
		// For now, just ensure it exists and is correct
		log.Info("Valkey secret already exists, updating if necessary", "SecretName", secretName)
		err = r.Update(ctx, secret)
		if err != nil {
			log.Error(err, "Failed to update valkey secret", "SecretName", secretName)
			return "", fmt.Errorf("failed to update valkey secret %s: %w", secretName, err)
		}
	} else {
		log.Info("Creating Valkey Kubernetes secret", "SecretName", secretName)
		err = r.Create(ctx, secret)
		if err != nil && !apierrors.IsAlreadyExists(err) {
			log.Error(err, "Failed to create valkey secret", "SecretName", secretName)
			return "", fmt.Errorf("failed to create valkey secret %s: %w", secretName, err)
		}
	}

	log.Info("Successfully ensured Valkey secret exists", "SecretName", secretName)
	return secretName, nil
}

// --- Helper functions for credential provisioning ---

// provisionPostgresCredentials generates credentials, creates a DB role, and creates a K8s Secret.
// Returns the name of the created Secret or an error.
func (r *AgentReconciler) provisionPostgresCredentials(ctx context.Context, agent *agentsv1alpha1.Agent) (string, error) {
	log := logf.FromContext(ctx).WithValues("agent", agent.Name, "namespace", agent.Namespace, "agentType", agent.Spec.Type)

	secretName := fmt.Sprintf("agent-%s-postgres-creds", agent.Spec.Type)

	// Use createRoleName for role names to remove dashes and underscores
	// This circumvents PostgreSQL's constraints for role names
	roleName := createRoleName(agent.Spec.Type)
	log.Info("Using role name with dashes and underscores removed", "OriginalType", agent.Spec.Type, "CleanedRoleName", roleName)
	dbUsername := fmt.Sprintf("agent_%s", roleName)

	// For schema name, we still use SanitizeForDbIdentifier which replaces special chars with underscores
	dbSchemaName := SanitizeForDbIdentifier(agent.Spec.Type)

	// 1. Generate Password
	password, err := generatePassword(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate password: %w", err)
	}

	// 2. Connect to Postgres using Operator's Admin Credentials
	//    These credentials are provided as individual environment variables
	pgUser := os.Getenv("POSTGRES_USER")
	pgPassword := os.Getenv("POSTGRES_PASSWORD")
	pgHost := os.Getenv("POSTGRES_HOST")
	pgPort := os.Getenv("POSTGRES_PORT")
	pgDB := os.Getenv("POSTGRES_DB")

	// Validate that we have all required connection parameters
	if pgUser == "" || pgPassword == "" || pgHost == "" || pgPort == "" || pgDB == "" {
		return "", fmt.Errorf("one or more required PostgreSQL environment variables are not set")
	}

	// Construct the connection string
	adminConnStr := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		pgUser, pgPassword, pgHost, pgPort, pgDB)

	db, err := sql.Open("postgres", adminConnStr)
	if err != nil {
		return "", fmt.Errorf("failed to connect to postgres as admin: %w", err)
	}
	defer db.Close()

	err = db.PingContext(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to ping postgres as admin: %w", err)
	}
	log.Info("Successfully connected to Postgres as admin")

	// 3. Create Role and Grant Permissions (Idempotent)
	//    Use transactions for atomicity
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if anything fails

	// Check if role exists
	var exists bool
	// Use $1 placeholder for pq driver
	err = tx.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)", dbUsername).Scan(&exists)
	if err != nil {
		return "", fmt.Errorf("failed to check if role %s exists: %w", dbUsername, err)
	}

	// Safely quote the identifier for use in SQL statements
	quotedDbUsername := pq.QuoteIdentifier(dbUsername)

	if !exists {
		log.Info("Creating database role", "RoleName", dbUsername)
		// Step 1: Create the role without password first
		_, err = tx.ExecContext(ctx, fmt.Sprintf("CREATE ROLE %s WITH LOGIN", quotedDbUsername))
		if err != nil {
			return "", fmt.Errorf("failed to create role %s: %w", dbUsername, err)
		}
		// Step 2: Set the password using ALTER ROLE and a parameter
		_, err = tx.ExecContext(ctx, fmt.Sprintf("ALTER ROLE %s WITH PASSWORD '%s'", quotedDbUsername, password))
		if err != nil {
			return "", fmt.Errorf("failed to set password for role %s: %w", dbUsername, err)
		}
		log.Info("Successfully created role and set password", "RoleName", dbUsername)
	} else {
		log.Info("Database role already exists, ensuring password is set", "RoleName", dbUsername)
		// If role exists, ensure the password is set (or updated if rotation logic is added)
		_, err = tx.ExecContext(ctx, fmt.Sprintf("ALTER ROLE %s WITH PASSWORD '%s'", quotedDbUsername, password))
		if err != nil {
			// Log the error but don't necessarily fail the whole provisioning if altering fails
			// This might happen due to permissions issues if the operator's role changed.
			log.Error(err, "Failed to alter role password, continuing", "RoleName", dbUsername)
			// return "", fmt.Errorf("failed to alter role %s password: %w", dbUsername, err)
		} else {
			log.Info("Successfully ensured password is set for existing role", "RoleName", dbUsername)
		}
	}
	// Grant agent user access to shared tables
	grantStmts := []string{
		fmt.Sprintf("GRANT SELECT, INSERT, UPDATE, DELETE ON public.agent_state TO %s", quotedDbUsername),
		fmt.Sprintf("GRANT SELECT, INSERT, UPDATE, DELETE ON public.agent_message_log TO %s", quotedDbUsername),
		fmt.Sprintf("GRANT SELECT, INSERT, UPDATE, DELETE ON public.agent_status TO %s", quotedDbUsername),
	}
	for _, grant := range grantStmts {
		_, err = tx.ExecContext(ctx, grant)
		if err != nil {
			log.Error(err, "Failed to grant privileges to agent user", "stmt", grant)
		}
	}

	// Grant CONNECT on the database
	dbName := os.Getenv("POSTGRES_DB") // Get the target DB name (should match what agents use)
	if dbName == "" {
		dbName = "agentbox" // Default if not set
	}
	log.Info("Granting CONNECT permission", "RoleName", dbUsername, "Database", dbName)
	_, err = tx.ExecContext(ctx, fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s", dbName, dbUsername))
	if err != nil {
		return "", fmt.Errorf("failed to grant connect on database %s to %s: %w", dbName, dbUsername, err)
	}

	// Create agent-specific schema and set ownership
	log.Info("Creating agent-specific schema", "SchemaName", dbSchemaName, "Owner", dbUsername)
	// Ensure schema name and owner name are safe identifiers before embedding in SQL
	// (SanitizeForDbIdentifier helps, but consider parameterization if complex names are possible)
	_, err = tx.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s AUTHORIZATION %s", dbSchemaName, dbUsername))
	if err != nil {
		return "", fmt.Errorf("failed to create schema %s for role %s: %w", dbSchemaName, dbUsername, err)
	}
	log.Info("Successfully created/ensured agent schema exists", "SchemaName", dbSchemaName)

	// Ensure common 'public' schema exists (idempotent)
	log.Info("Ensuring public schema exists")
	_, err = tx.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS public")
	if err != nil {
		return "", fmt.Errorf("failed to create public schema: %w", err)
	}

	// Grant usage on common 'public' schema
	log.Info("Granting USAGE on public schema", "RoleName", dbUsername)
	_, err = tx.ExecContext(ctx, fmt.Sprintf("GRANT USAGE ON SCHEMA public TO %s", dbUsername))
	if err != nil {
		return "", fmt.Errorf("failed to grant usage on schema public to %s: %w", err)
	}

	// Grant specific permissions on common tables/sequences in 'public' schema (DEFINE THESE!)
	// Example: Grant SELECT on a common 'config' table
	// _, err = tx.ExecContext(ctx, fmt.Sprintf("GRANT SELECT ON TABLE public.common_config TO %s", dbUsername))
	// if err != nil {
	// 	log.Error(err, "Failed to grant SELECT on public.common_config (table might not exist yet)", "RoleName", dbUsername)
	// }

	// Commit transaction
	if err = tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}
	log.Info("Successfully created/verified database role and permissions", "RoleName", dbUsername)

	// 4. Create/Update Kubernetes Secret
	// For the host, construct the FQDN with namespace to allow cross-namespace resolution
	pgNamespace := os.Getenv("OPERATOR_NAMESPACE") // Should be set in the deployment
	if pgNamespace == "" {
		// Default to the operator's own namespace if not explicitly set
		pgNamespace = "agentbox-system"
	}
	// Ensure we have the correct service name for PostgreSQL
	pgServiceName := os.Getenv("POSTGRES_HOST")
	if pgServiceName == "" {
		pgServiceName = "agentbox-postgresql" // Default service name if not set
	}

	// Always use fully qualified domain name (FQDN) for cross-namespace service access
	pgFQDN := fmt.Sprintf("%s.%s.svc.cluster.local", pgServiceName, pgNamespace)

	secretData := map[string][]byte{
		"username": []byte(dbUsername),
		"password": []byte(password),
		"database": []byte(dbName),                     // Include DB name for convenience
		"host":     []byte(pgFQDN),                     // Use FQDN with namespace
		"port":     []byte(os.Getenv("POSTGRES_PORT")), // Get port from env
	}
	// Ensure host/port env vars are set in operator deployment
	if string(secretData["host"]) == "" || string(secretData["port"]) == "" {
		log.Error(fmt.Errorf("POSTGRES_HOST or POSTGRES_PORT env vars not set for operator"), "cannot fully populate secret data")
		// Decide if this is fatal or if defaults should be used
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: agent.Namespace,
		},
		Data: secretData,
		Type: corev1.SecretTypeOpaque,
	}

	// Set owner reference so Secret is deleted when Agent is deleted
	if err := controllerutil.SetControllerReference(agent, secret, r.Scheme); err != nil {
		// Log but don't fail the whole operation just for owner ref
		log.Error(err, "Failed to set controller reference on secret", "SecretName", secretName)
	}

	// Try to create the secret
	log.Info("Creating Kubernetes secret", "SecretName", secretName)
	err = r.Create(ctx, secret)
	if err != nil && apierrors.IsAlreadyExists(err) {
		// Secret already exists, try to update it (e.g., if password needs rotation - though not implemented above)
		log.Info("Secret already exists, attempting update (currently no-op)", "SecretName", secretName)
		// Fetch existing
		existingSecret := &corev1.Secret{}
		if getErr := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: agent.Namespace}, existingSecret); getErr == nil {
			// Update data if needed (e.g., if password rotation was implemented)
			// existingSecret.Data = secretData
			// if updateErr := r.Update(ctx, existingSecret); updateErr != nil {
			// 	log.Error(updateErr, "Failed to update existing secret", "SecretName", secretName)
			// 	return "", fmt.Errorf("failed to update existing secret %s: %w", secretName, updateErr)
			// }
		} else {
			log.Error(getErr, "Failed to get existing secret for update", "SecretName", secretName)
			// Continue, maybe creation failed transiently before
		}
	} else if err != nil {
		// Other error during creation
		log.Error(err, "Failed to create secret", "SecretName", secretName)
		return "", fmt.Errorf("failed to create secret %s: %w", secretName, err)
	}

	log.Info("Successfully ensured Kubernetes secret exists", "SecretName", secretName)
	return secretName, nil
}

// generatePassword creates a random password string of specified length.
func generatePassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Use URLEncoding to avoid characters that might cause issues in connection strings or env vars
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// SanitizeForDbIdentifier replaces potentially problematic characters in a string
// intended for use as a PostgreSQL identifier (like schema or role name).
// Basic example: replace non-alphanumeric with underscore. Needs refinement for edge cases.
func SanitizeForDbIdentifier(input string) string {
	// Very basic sanitization - replace common problematic chars like '-' with '_'
	// A more robust solution might use regex or allowlisting.
	// Also consider lowercasing as Postgres identifiers are case-insensitive unless quoted.
	sanitized := ""
	for _, r := range input {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			sanitized += string(r)
		} else {
			sanitized += "_" // Replace other chars with underscore
		}
	}
	// Ensure it doesn't start with a number or underscore if that's a rule
	// Ensure it's not a reserved keyword
	// Consider max length
	return sanitized
}

// createRoleName strips dashes and underscores from the input string to create
// a valid PostgreSQL role name that circumvents PG's constraints.
func createRoleName(input string) string {
	// Remove dashes and underscores
	result := ""
	for _, r := range input {
		if r != '-' && r != '_' {
			result += string(r)
		}
	}
	return result
}

func ensureAgentTables(db *sql.DB, log logr.Logger) error {
	const maxAttempts = 20
	const delay = 5 * time.Second
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		stmts := []string{
			`CREATE TABLE IF NOT EXISTS public.agent_state (
			agent_id TEXT PRIMARY KEY,
			state_json JSONB,
			updated_at TIMESTAMPTZ
		)`,
			`CREATE TABLE IF NOT EXISTS public.agent_message_log (
			id UUID PRIMARY KEY,
			agent_id TEXT,
			direction TEXT,
			sender TEXT,
			target TEXT,
			payload JSONB,
			timestamp TIMESTAMPTZ
		)`,
			`CREATE TABLE IF NOT EXISTS public.agent_status (
			agent_id TEXT PRIMARY KEY,
			phase TEXT,
			message TEXT,
			step TEXT,
			updated_at TIMESTAMPTZ
		)`,
			// `CREATE TABLE IF NOT EXISTS public.agent_embeddings (
			// 	id UUID PRIMARY KEY,
			// 	agent_id TEXT,
			// 	embedding VECTOR, -- Replace VECTOR with the actual type used by your Postgres extension
			// 	metadata JSONB,
			// 	created_at TIMESTAMPTZ
			// )`,
		}
		for _, stmt := range stmts {
			if _, err := db.Exec(stmt); err != nil {
				log.Error(err, "Failed to create agent table", "stmt", stmt)
				lastErr = err
				break
			}
			lastErr = nil
		}
		if lastErr == nil {
			log.Info("Ensured all agent tables exist in Postgres")
			return nil
		}
		log.Info("Retrying agent table creation after delay", "attempt", attempt, "delay", delay)
		time.Sleep(delay)
	}
	return fmt.Errorf("failed to create agent tables after %d attempts: %w", maxAttempts, lastErr)
}

// SetupWithManager sets up the controller with the Manager.
func (r *AgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Ensure agent tables exist in Postgres at operator startup
	log := ctrl.Log.WithName("setup")
	pgUser := os.Getenv("POSTGRES_USER")
	pgPassword := os.Getenv("POSTGRES_PASSWORD")
	pgHost := os.Getenv("POSTGRES_HOST")
	pgPort := os.Getenv("POSTGRES_PORT")
	pgDB := os.Getenv("POSTGRES_DB")
	if pgUser != "" && pgPassword != "" && pgHost != "" && pgPort != "" && pgDB != "" {
		adminConnStr := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
			pgUser, pgPassword, pgHost, pgPort, pgDB)
		db, err := sql.Open("postgres", adminConnStr)
		if err == nil {
			defer db.Close()
			if err := ensureAgentTables(db, log); err != nil {
				log.Error(err, "Failed to ensure agent tables in Postgres")
			}
		} else {
			log.Error(err, "Failed to connect to Postgres for table creation")
		}
	} else {
		log.Info("Skipping agent table creation: missing Postgres admin env vars")
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&agentsv1alpha1.Agent{}).
		Owns(&corev1.Pod{}). // Watch Pods owned by Agent CRs
		Named("agent").
		Complete(r)
}
