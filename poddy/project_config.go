package poddy

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/dogboy21/poddy/models"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type CodeServerConfig struct {
	BaseImage  string   `yaml:"image"`
	Extensions []string `yaml:"extensions"`
}

type JbProjectorConfig struct {
}

type JbFleetConfig struct {
}

type ServiceConfig struct {
	Name string `yaml:"name"`
	Port uint16 `yaml:"port"`
}

type ProjectConfig struct {
	CodeServer  *CodeServerConfig  `yaml:"codeServer"`
	JbProjector *JbProjectorConfig `yaml:"jbProjector"`
	JbFleet     *JbFleetConfig     `yaml:"jbFleet"`

	Services []ServiceConfig `yaml:"services"`
}

func (p *ProjectConfig) createDeploymentSpec(project models.Project, user models.User, accessToken string) (*appsv1.DeploymentSpec, error) {
	if p.CodeServer == nil && p.JbProjector == nil && p.JbFleet == nil {
		p.CodeServer = &CodeServerConfig{}
	}

	mutuallyExclusive := !(p.CodeServer != nil && p.JbProjector != nil && p.JbFleet != nil) && (p.CodeServer != nil) != (p.JbProjector != nil) != (p.JbFleet != nil)
	if !mutuallyExclusive {
		return nil, errors.New("only one project type can be configured")
	}

	parsedCloneUrl, err := url.Parse(project.GetHttpCloneUrl())
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository clone url: %v", err)
	}

	if p.CodeServer != nil {
		envVars := []corev1.EnvVar{
			{
				Name:  "REPO_URL",
				Value: project.GetHttpCloneUrl(),
			},
			{
				Name:  "GIT_HOST",
				Value: parsedCloneUrl.Host,
			},
			{
				Name:  "USERNAME",
				Value: user.GetUsername(),
			},
			{
				Name:  "EMAIL",
				Value: user.GetEmail(),
			},
			{
				Name:  "ACCESS_TOKEN",
				Value: accessToken,
			},
		}

		codeServerImage := p.CodeServer.BaseImage
		if len(codeServerImage) == 0 {
			codeServerImage = "codercom/code-server:4.0.2"
		}

		workspaceSetupCommands := "set -v\n" +
			"echo -e \"machine $GIT_HOST\\nlogin oauth2\\npassword $ACCESS_TOKEN\" > ~/.netrc\n" +
			"chmod 600 ~/.netrc\n" +
			"git clone $REPO_URL /workspace\n" +
			"cp ~/.netrc /config/.netrc\n" +
			"echo -e \"[user]\\\\n        name = $USERNAME\\\\n        email = $EMAIL\" > /config/.gitconfig\n"

		extensionCommands := ""
		for _, extension := range p.CodeServer.Extensions {
			extensionCommands += fmt.Sprintf("/usr/bin/entrypoint.sh --install-extension %s\n", extension)
		}

		serverStartCommands := "set -v\n" +
			extensionCommands +
			"/usr/bin/entrypoint.sh --bind-addr 0.0.0.0:8080 --auth none /workspace\n"

		return &appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"workspace-type": "code-server",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "workspace-data",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "config-data",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:    "workspace-setup",
							Image:   "alpine/git:user",
							Env:     envVars,
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{workspaceSetupCommands},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "workspace-data",
									MountPath: "/workspace",
								},
								{
									Name:      "config-data",
									MountPath: "/config",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:    "code-server",
							Image:   codeServerImage,
							Env:     envVars,
							Command: []string{"/bin/sh", "-c"},
							Args:    []string{serverStartCommands},
							Ports: []corev1.ContainerPort{
								{
									Name:          "server",
									ContainerPort: 8080,
									Protocol:      "TCP",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "workspace-data",
									MountPath: "/workspace",
								},
								{
									Name:      "config-data",
									MountPath: "/home/coder/.netrc",
									SubPath:   ".netrc",
								},
								{
									Name:      "config-data",
									MountPath: "/home/coder/.gitconfig",
									SubPath:   ".gitconfig",
								},
							},
						},
					},
				},
			},
		}, nil
	} else if p.JbProjector != nil {

	} else if p.JbFleet != nil {

	}

	return nil, errors.New("unsupported project type")
}

func (p *ProjectConfig) getServicePorts() []corev1.ServicePort {
	ports := make([]corev1.ServicePort, len(p.Services)+1)
	ports[0] = corev1.ServicePort{
		Name:     "server",
		Protocol: "TCP",
		Port:     8080,
		TargetPort: intstr.IntOrString{
			StrVal: "server",
		},
	}

	for i := 0; i < len(p.Services); i++ {
		ports[i+1] = corev1.ServicePort{
			Name:     p.Services[i].Name,
			Protocol: "TCP",
			Port:     int32(p.Services[i].Port),
			TargetPort: intstr.IntOrString{
				IntVal: int32(p.Services[i].Port),
			},
		}
	}

	return ports
}

func (p *ProjectConfig) getIngressRules(workspaceName, ingressDomain string) []networkv1.IngressRule {
	ingressRules := make([]networkv1.IngressRule, len(p.Services)+1)
	ingressRules[0] = networkv1.IngressRule{
		Host: ingressDomain,
		IngressRuleValue: networkv1.IngressRuleValue{
			HTTP: &networkv1.HTTPIngressRuleValue{
				Paths: []networkv1.HTTPIngressPath{
					{
						Path:     "/",
						PathType: pathTypePointer(networkv1.PathTypePrefix),
						Backend: networkv1.IngressBackend{
							Service: &networkv1.IngressServiceBackend{
								Name: workspaceName,
								Port: networkv1.ServiceBackendPort{
									Name: "server",
								},
							},
						},
					},
				},
			},
		},
	}

	for i := 0; i < len(p.Services); i++ {
		ingressRules[i+1] = networkv1.IngressRule{
			Host: fmt.Sprintf("%s-%s", p.Services[i].Name, ingressDomain),
			IngressRuleValue: networkv1.IngressRuleValue{
				HTTP: &networkv1.HTTPIngressRuleValue{
					Paths: []networkv1.HTTPIngressPath{
						{
							Path:     "/",
							PathType: pathTypePointer(networkv1.PathTypePrefix),
							Backend: networkv1.IngressBackend{
								Service: &networkv1.IngressServiceBackend{
									Name: workspaceName,
									Port: networkv1.ServiceBackendPort{
										Number: int32(p.Services[i].Port),
									},
								},
							},
						},
					},
				},
			},
		}
	}

	return ingressRules
}
