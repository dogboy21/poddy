package poddy

import (
	"context"
	"fmt"

	"github.com/dogboy21/poddy/config"
	"github.com/dogboy21/poddy/models"
	petname "github.com/dustinkirkland/golang-petname"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func getKubernetesConfig() (*rest.Config, error) {
	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err := kubeConfig.ClientConfig()
		if err != nil {
			return nil, err
		}

		return config, nil
	}

	return inClusterConfig, nil
}

func int32Pointer(v int32) *int32 {
	return &v
}

func nonEmptyStringPointer(str string) *string {
	if len(str) == 0 {
		return nil
	}

	return &str
}

func pathTypePointer(pathType networkv1.PathType) *networkv1.PathType {
	return &pathType
}

func createWorkspace(provider models.RepositoryProvider, projectSlug, projectBranch string, currentUser models.User, accessToken string) (string, string, error) {
	project, err := provider.GetProject(projectSlug)
	if err != nil {
		return "", "", fmt.Errorf("failed to get project: %v", err)
	}

	if projectBranch == "" {
		projectBranch = project.GetDefaultBranch()
	}

	branchExists, err := provider.DoesProjectBranchExist(projectSlug, projectBranch)
	if err != nil {
		return "", "", fmt.Errorf("failed to find branch %s for project %s: %v", projectBranch, projectSlug, err)
	}
	if !branchExists {
		return "", "", fmt.Errorf("no branch found for name %s", projectBranch)
	}

	poddyProjectConfigFile, err := provider.GetProjectFile(projectSlug, projectBranch, ".poddy.yml")
	if err != nil {
		return "", "", fmt.Errorf("failed to get poddy config for project %s: %v", projectSlug, err)
	}

	var projectConfig ProjectConfig
	if err := yaml.Unmarshal(poddyProjectConfigFile, &projectConfig); err != nil {
		return "", "", fmt.Errorf("failed to parse poddy project config for %s: %v", projectSlug, err)
	}

	deploymentSpec, err := projectConfig.createDeploymentSpec(project, currentUser, accessToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to create deployment spec from project config: %v", err)
	}

	kubernetesConfig, err := getKubernetesConfig()
	if err != nil {
		return "", "", fmt.Errorf("failed to get Kubernetes config: %v", err)
	}

	clientSet, err := kubernetes.NewForConfig(kubernetesConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	workspaceName := petname.Generate(5, "-")

	labels := map[string]string{
		"managed-by":      "poddy",
		"workspace-name":  workspaceName,
		"workspace-owner": currentUser.GetUsername(),
	}

	deploymentSpec.Replicas = int32Pointer(1)

	deploymentSpec.Selector = &metav1.LabelSelector{
		MatchLabels: labels,
	}

	for k, v := range labels {
		deploymentSpec.Template.ObjectMeta.Labels[k] = v
	}

	deployment, err := clientSet.AppsV1().Deployments(config.DeploymentNamespace()).Create(context.Background(), &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workspaceName,
			Namespace: config.DeploymentNamespace(),
			Labels:    labels,
		},
		Spec: *deploymentSpec,
	}, metav1.CreateOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to create deployment: %v", err)
	}

	ownerReferences := []metav1.OwnerReference{
		{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       deployment.Name,
			UID:        deployment.UID,
		},
	}

	_, err = clientSet.CoreV1().Services(config.DeploymentNamespace()).Create(context.Background(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            workspaceName,
			Namespace:       config.DeploymentNamespace(),
			Labels:          labels,
			OwnerReferences: ownerReferences,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Type:     corev1.ServiceTypeClusterIP,
			Ports:    projectConfig.getServicePorts(),
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to create service for deployment: %v", err)
	}

	ingressDomain := fmt.Sprintf("%s.%s", workspaceName, config.DeploymentBaseDomain())

	_, err = clientSet.NetworkingV1().Ingresses(config.DeploymentNamespace()).Create(context.Background(), &networkv1.Ingress{ //TODO Ingress Annotations
		ObjectMeta: metav1.ObjectMeta{
			Name:            workspaceName,
			Namespace:       config.DeploymentNamespace(),
			Labels:          labels,
			OwnerReferences: ownerReferences,
		},
		Spec: networkv1.IngressSpec{
			IngressClassName: nonEmptyStringPointer(config.DeploymentIngressClass()),
			Rules:            projectConfig.getIngressRules(workspaceName, ingressDomain),
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to create ingress for deployment: %v", err)
	}

	return workspaceName, ingressDomain, nil
}

func listWorkspaces(currentUser models.User) ([]map[string]string, error) {
	kubernetesConfig, err := getKubernetesConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get Kubernetes config: %v", err)
	}

	clientSet, err := kubernetes.NewForConfig(kubernetesConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	deploymentList, err := clientSet.AppsV1().Deployments(config.DeploymentNamespace()).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("managed-by=poddy,workspace-owner=%s", currentUser.GetUsername()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %v", err)
	}

	workspaceList := make([]map[string]string, len(deploymentList.Items))
	for i := 0; i < len(workspaceList); i++ {
		workspaceName := deploymentList.Items[i].ObjectMeta.Labels["workspace-name"]
		workspaceList[i] = map[string]string{
			"name": workspaceName,
			"url":  fmt.Sprintf("%s.%s", workspaceName, config.DeploymentBaseDomain()),
		}
	}

	return workspaceList, nil
}

func deleteWorkspace(workspaceName string, currentUser models.User) error {
	kubernetesConfig, err := getKubernetesConfig()
	if err != nil {
		return fmt.Errorf("failed to get Kubernetes config: %v", err)
	}

	clientSet, err := kubernetes.NewForConfig(kubernetesConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %v", err)
	}

	deploymentList, err := clientSet.AppsV1().Deployments(config.DeploymentNamespace()).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("managed-by=poddy,workspace-owner=%s", currentUser.GetUsername()),
	})
	if err != nil {
		return fmt.Errorf("failed to list deployments: %v", err)
	}

	if len(deploymentList.Items) == 0 {
		return fmt.Errorf("failed to find workspace %s", workspaceName)
	}

	return clientSet.AppsV1().Deployments(config.DeploymentNamespace()).Delete(context.Background(), workspaceName, metav1.DeleteOptions{})
}
