package tests

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const (
	namespaceEnvVarName                     = "TEST_NAMESPACE"
	imageEnvVarName                         = "TEST_CHECKUP_IMAGE"
	vmUnderTestContainerDiskImageEnvVarName = "VM_CONTAINER_DISK_IMAGE_URL"
)

const (
	defaultNamespace = "kiagnose-demo"
	defaultImageName = "quay.io/kiagnose/kubevirt-realtime-checkup:main"
)

var (
	client                        *kubernetes.Clientset
	testNamespace                 string
	testImageName                 string
	vmUnderTestContainerDiskImage string
)

func TestTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Tests Suite")
}

var _ = BeforeSuite(func() {
	kubeConfigPath := os.Getenv("KUBECONFIG")
	if kubeConfigPath == "" {
		home := homedir.HomeDir()
		kubeConfigPath = filepath.Join(home, ".kube", "config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	Expect(err).NotTo(HaveOccurred())

	client, err = kubernetes.NewForConfig(config)
	Expect(err).NotTo(HaveOccurred())

	if testNamespace = os.Getenv(namespaceEnvVarName); testNamespace == "" {
		testNamespace = defaultNamespace
	}

	if testImageName = os.Getenv(imageEnvVarName); testImageName == "" {
		testImageName = defaultImageName
	}

	vmUnderTestContainerDiskImage = os.Getenv(vmUnderTestContainerDiskImageEnvVarName)
})
