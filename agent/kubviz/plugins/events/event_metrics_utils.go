package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	kubevizotel "github.com/intelops/kubviz/agent/kubviz/otel"
	"github.com/intelops/kubviz/constants"
	"github.com/intelops/kubviz/model"
	"github.com/intelops/kubviz/pkg/opentelemetry"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var ClusterName string = os.Getenv("CLUSTER_NAME")

// publishMetrics publishes stream of events
// with subject "METRICS.created"
func PublishMetrics(clientset *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, restMapper meta.RESTMapper, js nats.JetStreamContext, errCh chan error) {

	ctx := context.Background()
	tracer := otel.Tracer("kubviz-publish-metrics")
	_, span := tracer.Start(opentelemetry.BuildContext(ctx), "publishMetrics")
	span.SetAttributes(attribute.String("kubviz-agent", "publish-metrics"))
	defer span.End()

	watchK8sEvents(clientset, dynamicClient, restMapper, js)
	errCh <- nil
}

func publishK8sMetrics(id string, mtype string, mdata *v1.Event, js nats.JetStreamContext, imageName string) (bool, error) {

	ctx := context.Background()
	tracer := otel.Tracer("kubviz-publish-k8smetrics")
	_, span := tracer.Start(opentelemetry.BuildContext(ctx), "publishK8sMetrics")
	span.SetAttributes(attribute.String("kubviz-agent", "publish-k8smetrics"))
	defer span.End()

	metrics := model.Metrics{
		ID:          id,
		Type:        mtype,
		Event:       mdata,
		ClusterName: ClusterName,
		ImageName:   imageName,
	}
	metricsJson, _ := json.Marshal(metrics)
	_, err := js.Publish(constants.EventSubject, metricsJson)

	if err != nil {
		return true, err
	}
	log.Printf("Metrics with ID:%s has been published\n", id)
	return false, nil
}

func getK8sPodImages(clientset *kubernetes.Clientset, namespace, podName string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var images []string
	for _, container := range pod.Spec.Containers {
		images = append(images, container.Image)
	}

	if len(images) == 0 {
		return nil, errors.New("no containers found in the pod")
	}

	return images, nil
}

// createStream creates a stream by using JetStreamContext
func CreateStream(js nats.JetStreamContext) error {
	// Check if the METRICS stream already exists; if not, create it.
	stream, err := js.StreamInfo(constants.StreamName)
	log.Printf("Retrieved stream %s", fmt.Sprintf("%v", stream))
	if err != nil {
		log.Printf("Error getting stream %s", err)
	}
	if stream == nil {
		log.Printf("creating stream %q and subjects %q", constants.StreamName, constants.StreamSubjects)
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     constants.StreamName,
			Subjects: []string{constants.StreamSubjects},
		})
		CheckErr(err)
	}
	return nil

}

func GetK8sClient(config *rest.Config) *kubernetes.Clientset {
	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	CheckErr(err)
	return clientset
}

func GetK8sPods(clientset *kubernetes.Clientset) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	CheckErr(err)
	var sb strings.Builder
	for i, pod := range pods.Items {
		sb.WriteString("Name-" + strconv.Itoa(i) + ": ")
		sb.WriteString(pod.Name)
		sb.WriteString("   ")
		sb.WriteString("Namespace-" + strconv.Itoa(i) + ": ")
		sb.WriteString(pod.Namespace)
		sb.WriteString("   ")
	}
	return sb.String()
}

func GetK8sNodes(clientset *kubernetes.Clientset) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	CheckErr(err)
	var sb strings.Builder
	for i, node := range nodes.Items {
		sb.WriteString("Name-" + strconv.Itoa(i) + ": ")
		sb.WriteString(node.Name)
	}
	return sb.String()
}

func GetK8sEvents(clientset *kubernetes.Clientset) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	events, err := clientset.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	CheckErr(err)
	j, err := json.MarshalIndent(events, "", "  ")
	CheckErr(err)
	log.Printf("%#v", string(j))
	return string(j)
}

func CheckErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
func LogErr(err error) {
	if err != nil {
		log.Println(err)
	}
}
func watchK8sEvents(clientset *kubernetes.Clientset, dynamicClient *dynamic.DynamicClient, restMapper meta.RESTMapper, js nats.JetStreamContext) {

	ctx := context.Background()
	tracer := otel.Tracer("kubviz-watch-k8sevents")
	_, span := tracer.Start(opentelemetry.BuildContext(ctx), "watchK8sEvents")
	span.SetAttributes(attribute.String("kubviz-agent", "watch-k8sevents"))
	defer span.End()

	watchlist := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"events",
		v1.NamespaceAll,
		fields.Everything(),
	)

	_, controller := cache.NewInformer(
		watchlist,
		&v1.Event{},
		0, // Duration is int64
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				event := obj.(*v1.Event)
				images, err := getK8sPodImages(clientset, event.InvolvedObject.Namespace, event.InvolvedObject.Name)
				if err != nil {
					log.Println("Error retrieving image names:", err)
					return
				}
				for _, image := range images {
					publishK8sMetrics(string(event.ObjectMeta.UID), "ADD", event, js, image)
				}

				// Publish to OTEL logs
				unstructuredObj, err := getUnstructuredObject(dynamicClient, restMapper, event.InvolvedObject)
				if err != nil {
					log.Printf("Error getting unstructured object: %v\n", err)
					return
				}

				err = publishUnstructuredObject(unstructuredObj)
				if err != nil {
					log.Printf("Error publishing unstructured object as OTEL log: %v\n", err)
					return
				}

				// fmt.Println("watchK8sEvents AddFunc:")
				// fmt.Println(unstructuredObj)
			},
			DeleteFunc: func(obj interface{}) {
				event := obj.(*v1.Event)
				images, err := getK8sPodImages(clientset, event.InvolvedObject.Namespace, event.InvolvedObject.Name)
				if err != nil {
					log.Println("Error retrieving image names:", err)
					return
				}
				for _, image := range images {
					publishK8sMetrics(string(event.ObjectMeta.UID), "DELETE", event, js, image)
				}
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				event := newObj.(*v1.Event)
				images, err := getK8sPodImages(clientset, event.InvolvedObject.Namespace, event.InvolvedObject.Name)
				if err != nil {
					log.Println("Error retrieving image names:", err)
					return
				}
				for _, image := range images {
					publishK8sMetrics(string(event.ObjectMeta.UID), "UPDATE", event, js, image)
				}
			},
		},
	)
	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(stop)

	for {
		time.Sleep(time.Second)
	}
}

func getUnstructuredObject(dynamicClient dynamic.Interface, restMapper meta.RESTMapper, involvedObject v1.ObjectReference) (*unstructured.Unstructured, error) {
	gvk := schema.FromAPIVersionAndKind(involvedObject.APIVersion, involvedObject.Kind)

	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get REST mapping: %v", err)
	}

	gvr := mapping.Resource

	return dynamicClient.Resource(gvr).Namespace(involvedObject.Namespace).Get(context.TODO(), involvedObject.Name, metav1.GetOptions{})
}

func publishUnstructuredObject(obj *unstructured.Unstructured) error {
	objJson, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	kubevizotel.PublishEventLog(constants.EventSubject_kubeallresources, objJson)
	return nil
}
