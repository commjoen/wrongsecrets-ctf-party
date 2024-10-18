package testutil

import (
	"log"
	"os"

	"github.com/juice-shop/multi-juicer/balancer/pkg/bundle"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes/fake"
)

func NewTestBundle() *bundle.Bundle {
	clientset := fake.NewSimpleClientset()

	return &bundle.Bundle{
		ClientSet:             clientset,
		StaticAssetsDirectory: "../ui/build/",
		RuntimeEnvironment: bundle.RuntimeEnvironment{
			Namespace: "test-namespace",
		},
		Log: log.New(os.Stdout, "", log.LstdFlags),
		Config: &bundle.Config{
			JuiceShopConfig: bundle.JuiceShopConfig{
				ImagePullPolicy: "IfNotPresent",
				Image:           "bkimminich/juice-shop",
				Tag:             "latest",
				NodeEnv:         "multi-juicer",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("200m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
				},
			},
			CookieConfig: bundle.CookieConfig{
				SigningKey: "test-signing-key",
			},
		},
	}
}
