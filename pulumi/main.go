package main

import (
	"github.com/pulumi/pulumi-gcp/sdk/v4/go/gcp/container"
	"github.com/pulumi/pulumi-gcp/sdk/v4/go/gcp/projects"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi"
)

var location = "us-east1"
var k8sVersion = ""
var minNodeCount = 1
var maxNodeCount = 3
var preemptible = true
var machineType = "e2-standard-2"

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		containerService, err := projects.NewService(ctx, "project", &projects.ServiceArgs{
			Service: pulumi.String("container.googleapis.com"),
		})
		if err != nil {
			return err
		}

		if len(k8sVersion) == 0 {
			engineVersions, err := container.GetEngineVersions(ctx, &container.GetEngineVersionsArgs{
				Location: &location,
			})
			if err != nil {
				return err
			}
			k8sVersion = engineVersions.LatestMasterVersion
		}
		cluster, err := container.NewCluster(ctx, "primary", &container.ClusterArgs{
			Name:                  pulumi.StringPtr("devops-toolkit-pulumi"),
			Location:              pulumi.StringPtr(location),
			MinMasterVersion:      pulumi.StringPtr(k8sVersion),
			RemoveDefaultNodePool: pulumi.BoolPtr(true),
			InitialNodeCount:      pulumi.Int(1),
		}, pulumi.DependsOn([]pulumi.Resource{containerService}))
		if err != nil {
			return err
		}

		_, err = container.NewNodePool(ctx, "primary_nodes", &container.NodePoolArgs{
			Name:             pulumi.StringPtr("devops-toolkit-pulumi"),
			Cluster:          cluster.Name,
			Location:         pulumi.StringPtr(location),
			Version:          pulumi.StringPtr(k8sVersion),
			InitialNodeCount: pulumi.IntPtr(minNodeCount),
			NodeConfig: &container.NodePoolNodeConfigArgs{
				Preemptible: pulumi.BoolPtr(preemptible),
				MachineType: pulumi.StringPtr(machineType),
				OauthScopes: pulumi.StringArray{
					pulumi.String("https://www.googleapis.com/auth/cloud-platform"),
				},
			},
			Autoscaling: &container.NodePoolAutoscalingArgs{
				MinNodeCount: pulumi.Int(minNodeCount),
				MaxNodeCount: pulumi.Int(maxNodeCount),
			},
			Management: &container.NodePoolManagementArgs{
				AutoUpgrade: pulumi.BoolPtr(false),
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("kubeconfig", generateKubeconfig(cluster.Endpoint, cluster.Name, cluster.MasterAuth))

		// k8sProvider, err := providers.NewProvider(ctx, "k8sprovider", &providers.ProviderArgs{
		// 	Kubeconfig: generateKubeconfig(cluster.Endpoint, cluster.Name, cluster.MasterAuth),
		// }, pulumi.DependsOn([]pulumi.Resource{cluster}))
		// if err != nil {
		// 	return err
		// }

		// appLabels := pulumi.StringMap{
		// 	"app": pulumi.String("devops-toolkit"),
		// }
		// _, err = appsv1.NewDeployment(ctx, "app-dep", &appsv1.DeploymentArgs{
		// 	Metadata: &metav1.ObjectMetaArgs{
		// 		Namespace: pulumi.StringPtr("default"),
		// 	},
		// 	Spec: appsv1.DeploymentSpecArgs{
		// 		Selector: &metav1.LabelSelectorArgs{
		// 			MatchLabels: appLabels,
		// 		},
		// 		Replicas: pulumi.Int(3),
		// 		Template: &corev1.PodTemplateSpecArgs{
		// 			Metadata: &metav1.ObjectMetaArgs{
		// 				Labels: appLabels,
		// 			},
		// 			Spec: &corev1.PodSpecArgs{
		// 				Containers: corev1.ContainerArray{
		// 					corev1.ContainerArgs{
		// 						Name:  pulumi.String("devops-toolkit"),
		// 						Image: pulumi.String("vfarcic/devops-toolkit-series"),
		// 					}},
		// 			},
		// 		},
		// 	},
		// }, pulumi.Provider(k8sProvider))
		// if err != nil {
		// 	return err
		// }

		return nil
	})
}

func generateKubeconfig(clusterEndpoint pulumi.StringOutput, clusterName pulumi.StringOutput,
	clusterMasterAuth container.ClusterMasterAuthOutput) pulumi.StringOutput {
	context := pulumi.Sprintf("demo_%s", clusterName)

	return pulumi.Sprintf(`apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: %s
    server: https://%s
  name: %s
contexts:
- context:
    cluster: %s
    user: %s
  name: %s
current-context: %s
kind: Config
preferences: {}
users:
- name: %s
  user:
    auth-provider:
      config:
        cmd-args: config config-helper --format=json
        cmd-path: gcloud
        expiry-key: '{.credential.token_expiry}'
        token-key: '{.credential.access_token}'
      name: gcp`,
		clusterMasterAuth.ClusterCaCertificate().Elem(),
		clusterEndpoint, context, context, context, context, context, context)
}
