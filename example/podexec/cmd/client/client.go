package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"

	"k8s.io/client-go/kubernetes/scheme"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"

	v1 "sigs.k8s.io/apiserver-builder-alpha/example/podexec/pkg/apis/podexec/v1"
)

var (
	SchemeGroupVersion = schema.GroupVersion{Group: "podexec.example.com", Version: "v1"}
)

func kubeRestConfig() (*restclient.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	overrides := &clientcmd.ConfigOverrides{ClusterDefaults: clientcmd.ClusterDefaults}
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	restConfig.APIPath = "/apis"
	restConfig.GroupVersion = &SchemeGroupVersion
	restConfig.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}

	return restConfig, nil
}

func ExecPod(podCmd string, cmdStdout io.Writer, cmdStderr io.Writer) error {
	podName := "pod-example"

	restConfig, err := kubeRestConfig()
	if err != nil {
		return err
	}

	restClient, err := restclient.RESTClientFor(restConfig)
	if err != nil {
		return err
	}

	podexec := &v1.PodExec{
		Container: podName,
		Command:   []string{"/bin/sh", "-c", podCmd},
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}

	req := restClient.Post().
		Resource("pods").
		Name(podName).
		Namespace("default").
		SubResource("exec").
		Param("container", podName).
		VersionedParams(podexec, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("NewSPDYExecutor: %w", err)
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: cmdStdout,
		Stderr: cmdStderr,
		Tty:    true,
	})
	return fmt.Errorf("ExecPod: %w", err)
}

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	err := ExecPod("ping -c 5 8.8.8.8", os.Stdout, os.Stderr)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}

	<-c
}
