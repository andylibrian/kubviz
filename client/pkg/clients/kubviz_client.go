package clients

import (
	"context"
	"encoding/json"
	"log"

	"github.com/intelops/kubviz/constants"
	"github.com/intelops/kubviz/model"
	"github.com/intelops/kubviz/pkg/opentelemetry"
	"github.com/kelseyhightower/envconfig"
	"github.com/kuberhealthy/kuberhealthy/v2/pkg/health"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/intelops/kubviz/client/pkg/clickhouse"
	"github.com/intelops/kubviz/client/pkg/config"
	"github.com/intelops/kubviz/client/pkg/dgraph"
	"github.com/intelops/kubviz/client/pkg/kubernetes"
	"go.opentelemetry.io/collector/pdata/plog"

	"github.com/dgraph-io/dgo/v240"
	"github.com/dgraph-io/dgo/v240/protos/api"
	"google.golang.org/grpc"
)

type SubscriptionInfo struct {
	Subject  string
	Consumer string
	Handler  func(msg *nats.Msg)
}

func (n *NATSContext) SubscribeAllKubvizNats(conn clickhouse.DBInterface) {

	ctx := context.Background()
	tracer := otel.Tracer("kubviz-client")
	_, span := tracer.Start(opentelemetry.BuildContext(ctx), "SubscribeAllKubvizNats")
	span.SetAttributes(attribute.String("kubviz-subscribe", "subscribe"))
	defer span.End()
	cfg := &config.Config{}
	if err := envconfig.Process("", cfg); err != nil {
		log.Fatalf("Could not parse env Config: %v", err)
	}

	plogUnmarshaller := &plog.JSONUnmarshaler{}

	// K8s Dgraph
	// TODO: export to config
	dgraphGrpcEndpoint := "dgraph-public:9080"

	// Initialize Dgraph client
	grpcConn, err := grpc.Dial(dgraphGrpcEndpoint, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Error connecting to Dgraph: %v", err)
	}
	defer conn.Close()
	dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(grpcConn))

	// Setup schema
	err = dgraph.SetupDgraphSchema(ctx, dgraphClient)
	if err != nil {
		log.Fatalf("Failed to set up Dgraph schema: %v", err)
	}

	// Create a new repository instance
	repo := dgraph.NewDgraphKubernetesResourceRepository(dgraphClient)
	relationshipRepo := dgraph.NewDgraphRelationshipRepository(dgraphClient)

	subscriptions := []SubscriptionInfo{
		{
			Subject:  constants.KetallSubject,
			Consumer: cfg.KetallConsumer,
			Handler: func(msg *nats.Msg) {
				msg.Ack()
				var metrics model.Resource
				err := json.Unmarshal(msg.Data, &metrics)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("Ketall Metrics Received: %#v,", metrics)
				conn.InsertKetallEvent(metrics)
				log.Println()
			},
		},
		{
			Subject:  constants.RakeesSubject,
			Consumer: cfg.RakeesConsumer,
			Handler: func(msg *nats.Msg) {
				msg.Ack()
				var metrics model.RakeesMetrics
				err := json.Unmarshal(msg.Data, &metrics)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("Rakees Metrics Received: %#v,", metrics)
				conn.InsertRakeesMetrics(metrics)
				log.Println()
			},
		},
		{
			Subject:  constants.KUBERHEALTHY_SUBJECT,
			Consumer: cfg.KuberhealthyConsumer,
			Handler: func(msg *nats.Msg) {
				msg.Ack()
				var metrics health.State
				err := json.Unmarshal(msg.Data, &metrics)
				if err != nil {
					log.Println("failed to unmarshal from nats", err)
					return
				}
				log.Printf("Kuberhealthy Metrics Received: %#v,", metrics)
				conn.InsertKuberhealthyMetrics(metrics)
				log.Println()
			},
		},
		{
			Subject:  constants.OutdatedSubject,
			Consumer: cfg.OutdatedConsumer,
			Handler: func(msg *nats.Msg) {
				msg.Ack()
				var metrics model.CheckResultfinal
				err := json.Unmarshal(msg.Data, &metrics)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("Outdated Metrics Received: %#v,", metrics)
				conn.InsertOutdatedEvent(metrics)
				log.Println()
			},
		},
		{
			Subject:  constants.DeprecatedSubject,
			Consumer: cfg.DeprecatedConsumer,
			Handler: func(msg *nats.Msg) {
				msg.Ack()
				var metrics model.DeprecatedAPI
				err := json.Unmarshal(msg.Data, &metrics)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("Deprecated API Metrics Received: %#v,", metrics)
				conn.InsertDeprecatedAPI(metrics)
				log.Println()
			},
		},
		{
			Subject:  constants.DeletedSubject,
			Consumer: cfg.DeletedConsumer,
			Handler: func(msg *nats.Msg) {
				msg.Ack()
				var metrics model.DeletedAPI
				err := json.Unmarshal(msg.Data, &metrics)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("Deleted API Metrics Received: %#v,", metrics)
				conn.InsertDeletedAPI(metrics)
				log.Println()
			},
		},
		{
			Subject:  constants.TRIVY_IMAGE_SUBJECT,
			Consumer: cfg.TrivyImageConsumer,
			Handler: func(msg *nats.Msg) {
				msg.Ack()
				var metrics model.TrivyImage
				err := json.Unmarshal(msg.Data, &metrics)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("Trivy Metrics Received: %#v,", metrics)
				conn.InsertTrivyImageMetrics(metrics)
				log.Println()
			},
		},
		{
			Subject:  constants.TRIVY_SBOM_SUBJECT,
			Consumer: cfg.TrivySbomConsumer,
			Handler: func(msg *nats.Msg) {
				msg.Ack()
				var metrics model.Sbom
				err := json.Unmarshal(msg.Data, &metrics)
				if err != nil {
					log.Println("failed to unmarshal from nats", err)
					return
				}
				log.Printf("Trivy sbom Metrics Received: %#v,", metrics)
				conn.InsertTrivySbomMetrics(metrics)
				log.Println()
			},
		},
		{
			Subject:  constants.KubvizSubject,
			Consumer: cfg.KubvizConsumer,
			Handler: func(msg *nats.Msg) {
				msg.Ack()
				var metrics model.Metrics
				err := json.Unmarshal(msg.Data, &metrics)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("Kubviz Metrics Received: %#v,", metrics)
				conn.InsertKubvizEvent(metrics)
				log.Println()
			},
		},
		{
			Subject:  constants.KUBESCORE_SUBJECT,
			Consumer: cfg.KubscoreConsumer,
			Handler: func(msg *nats.Msg) {
				msg.Ack()
				var metrics model.KubeScoreRecommendations
				err := json.Unmarshal(msg.Data, &metrics)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("Kubscore Metrics Received: %#v,", metrics)
				conn.InsertKubeScoreMetrics(metrics)
				log.Println()
			},
		},
		{
			Subject:  constants.TRIVY_K8S_SUBJECT,
			Consumer: cfg.TrivyConsumer,
			Handler: func(msg *nats.Msg) {
				msg.Ack()
				var metrics model.Trivy
				err := json.Unmarshal(msg.Data, &metrics)
				if err != nil {
					log.Fatal(err)
				}
				log.Printf("Trivy Metrics Received: %#v,", metrics)
				conn.InsertTrivyMetrics(metrics)
				log.Println()
			},
		},
		{
			Subject:  constants.KubeAllResourcesSubject,
			Consumer: cfg.KubeAllResourcesConsumer,
			Handler: func(msg *nats.Msg) {
				msg.Ack()

				logs, err := plogUnmarshaller.UnmarshalLogs(msg.Data)
				if err != nil {
					// logger.WithError(err).Error("error while unmarshalling logs")
					log.Println("error while unmarshalling logs")
					return
				}

				rls := logs.ResourceLogs()
				for i := 0; i < rls.Len(); i++ {
					rl := rls.At(i)
					sls := rl.ScopeLogs()
					for j := 0; j < sls.Len(); j++ {
						sl := sls.At(j)
						logRecords := sl.LogRecords()
						for k := 0; k < logRecords.Len(); k++ {
							logRecord := logRecords.At(k)

							// Print log body
							logRecordBody := logRecord.Body().AsString()

							// Print attributes
							// attrs := logRecord.Attributes()
							// attrs.Range(func(k string, v pcommon.Value) bool {
							// 	fmt.Printf("Attribute - %s: %v\n", k, v.AsString())
							// 	return true
							// })
							var k8sUnstructured unstructured.Unstructured
							err := json.Unmarshal([]byte(logRecordBody), &k8sUnstructured)
							if err != nil {
								log.Println(err)
								continue
							}

							// log.Printf("NATSContext.KubeAllResourcesConsumer: Received k8s unstructured: %#v,", logRecordBody)
							// log.Printf("NATSContext.KubeAllResourcesConsumer: Received k8s object: kind:%s, namespace:%s, name:%s\n\n", k8sUnstructured.GetKind(), k8sUnstructured.GetNamespace(), k8sUnstructured.GetName())
							// log.Println("-----")

							dgraphResource, err := kubernetes.ConvertUnstructuredToDgraphResource(&k8sUnstructured)
							if err != nil {
								log.Println(err)
								continue
							}

							// Save to Dgraph
							err = repo.Save(ctx, dgraphResource)
							if err != nil {
								log.Println(err)
								continue
							}

							// Debug
							// dgraphResourceJSON, _ := json.Marshal(dgraphResource)
							// log.Printf("Saved resource to Dgraph: %s\n", string(dgraphResourceJSON))
							// log.Println("-----")

							relationships := kubernetes.BuildRelationships(&k8sUnstructured)
							err = relationshipRepo.Save(ctx, dgraphResource.MetadataUID, relationships)
							if err != nil {
								log.Println("Failed to update relationships", err)
								continue
							}

							// Debug
							// relationshipsJSON, _ := json.Marshal(relationships)
							// log.Printf("Saved relationships to Dgraph: %s\n", string(relationshipsJSON))
						}
					}
				}
			},
		},
	}

	for _, sub := range subscriptions {
		log.Printf("Creating nats consumer %s with subject: %s \n", sub.Consumer, sub.Subject)
		n.stream.Subscribe(sub.Subject, sub.Handler, nats.Durable(sub.Consumer), nats.ManualAck())
	}
}
