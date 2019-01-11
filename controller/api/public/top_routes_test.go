package public

import (
	"context"
	"sort"
	"testing"

	"github.com/golang/protobuf/proto"
	pb "github.com/linkerd/linkerd2/controller/gen/public"
	pkgK8s "github.com/linkerd/linkerd2/pkg/k8s"
	"github.com/prometheus/common/model"
)

var booksConfig = []string{
	// deployment/books
	`kind: Deployment
apiVersion: apps/v1beta2
metadata:
  name: books
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: books
  template:
    metadata:
      labels:
        app: books
    spec:
      dnsPolicy: ClusterFirst
      containers:
      - image: buoyantio/booksapp:v0.0.2`,

	// service/books
	`apiVersion: v1
kind: Service
metadata:
  name: books
  namespace: default
spec:
  selector:
    app: books`,

	// po/books-64c68d6d46-jrmmx
	`apiVersion: v1
kind: Pod
metadata:
  labels:
    app: books
  name: books-64c68d6d46-jrmmx
  namespace: default
spec:
  containers:
  - image: buoyantio/booksapp:v0.0.2
status:
  phase: Running`,

	// serviceprofile/books.default.svc.cluster.local
	`apiVersion: linkerd.io/v1alpha1
kind: ServiceProfile
metadata:
  name: books.default.svc.cluster.local
  namespace: linkerd
spec:
  routes:
  - condition:
      method: GET
      pathRegex: /a
    name: /a
`,
}

type topRoutesExpected struct {
	expectedStatRPC
	req              pb.TopRoutesRequest  // the request we would like to test
	expectedResponse pb.TopRoutesResponse // the routes response we expect
}

func genEmptyTopRoutesResponse() pb.TopRoutesResponse {
	return pb.TopRoutesResponse{
		Response: &pb.TopRoutesResponse_Routes{
			Routes: &pb.RouteTable{
				Rows: nil,
			},
		},
	}
}

func routesMetric(routes []string) model.Vector {
	samples := make(model.Vector, 0)
	for _, route := range routes {
		samples = append(samples, genRouteSample(route))
	}
	samples = append(samples, genDefaultRouteSample())
	return samples
}

func genRouteSample(route string) *model.Sample {
	return &model.Sample{
		Metric: model.Metric{
			"rt_route":       model.LabelValue(route),
			"dst":            "books.default.svc.cluster.local",
			"classification": "success",
		},
		Value:     123,
		Timestamp: 456,
	}
}

func genDefaultRouteSample() *model.Sample {
	return &model.Sample{
		Metric: model.Metric{
			"dst":            "books.default.svc.cluster.local",
			"classification": "success",
		},
		Value:     123,
		Timestamp: 456,
	}
}

func testTopRoutes(t *testing.T, expectations []topRoutesExpected) {
	for _, exp := range expectations {

		mockProm, fakeGrpcServer, err := newMockGrpcServer(exp.expectedStatRPC)
		if err != nil {
			t.Fatalf("Error creating mock grpc server: %s", err)
		}

		rsp, err := fakeGrpcServer.TopRoutes(context.TODO(), &exp.req)
		if err != exp.err {
			t.Fatalf("Expected error: %s, Got: %s", exp.err, err)
		}

		err = exp.verifyPromQueries(mockProm)
		if err != nil {
			t.Fatal(err)
		}

		rows := rsp.GetRoutes().Rows

		if len(rows) != len(exp.expectedResponse.GetRoutes().Rows) {
			t.Fatalf(
				"Expected [%d] rows, got [%d].\nExpected:\n%s\nGot:\n%s",
				len(exp.expectedResponse.GetRoutes().Rows),
				len(rows),
				exp.expectedResponse.GetRoutes().Rows,
				rows,
			)
		}

		sort.Slice(rows, func(i, j int) bool {
			return rows[i].GetAuthority()+rows[i].GetRoute() < rows[j].GetAuthority()+rows[j].GetRoute()
		})

		for i, row := range rows {
			expected := exp.expectedResponse.GetRoutes().Rows[i]
			if !proto.Equal(row, expected) {
				t.Fatalf("Expected: %+v\n Got: %+v", expected, row)
			}
		}
	}
}

func TestTopRoutes(t *testing.T) {
	t.Run("Successfully performs a routes query", func(t *testing.T) {
		routes := []string{"/a"}
		counts := []uint64{123}
		expectations := []topRoutesExpected{
			topRoutesExpected{
				expectedStatRPC: expectedStatRPC{
					err:              nil,
					mockPromResponse: routesMetric([]string{"/a"}),
					expectedPrometheusQueries: []string{
						`histogram_quantile(0.5, sum(irate(route_response_latency_ms_bucket{deployment="books", direction="inbound", namespace="default"}[1m])) by (le, dst, rt_route))`,
						`histogram_quantile(0.95, sum(irate(route_response_latency_ms_bucket{deployment="books", direction="inbound", namespace="default"}[1m])) by (le, dst, rt_route))`,
						`histogram_quantile(0.99, sum(irate(route_response_latency_ms_bucket{deployment="books", direction="inbound", namespace="default"}[1m])) by (le, dst, rt_route))`,
						`sum(increase(route_response_total{deployment="books", direction="inbound", namespace="default"}[1m])) by (rt_route, dst, classification)`,
					},
					k8sConfigs: booksConfig,
				},
				req: pb.TopRoutesRequest{
					Selector: &pb.ResourceSelection{
						Resource: &pb.Resource{
							Namespace: "default",
							Type:      pkgK8s.Deployment,
							Name:      "books",
						},
					},
					TimeWindow: "1m",
					Outbound: &pb.TopRoutesRequest_None{
						None: &pb.Empty{},
					},
				},
				expectedResponse: GenTopRoutesResponse(routes, counts, "books"),
			},
		}

		testTopRoutes(t, expectations)
	})

	t.Run("Successfully performs a routes query for a service", func(t *testing.T) {
		routes := []string{"/a"}
		counts := []uint64{123}
		expectations := []topRoutesExpected{
			topRoutesExpected{
				expectedStatRPC: expectedStatRPC{
					err:              nil,
					mockPromResponse: routesMetric([]string{"/a"}),
					expectedPrometheusQueries: []string{
						`histogram_quantile(0.5, sum(irate(route_response_latency_ms_bucket{direction="inbound", dst=~"(books.default.svc.cluster.local)(:\\d+)?"}[1m])) by (le, dst, rt_route))`,
						`histogram_quantile(0.95, sum(irate(route_response_latency_ms_bucket{direction="inbound", dst=~"(books.default.svc.cluster.local)(:\\d+)?"}[1m])) by (le, dst, rt_route))`,
						`histogram_quantile(0.99, sum(irate(route_response_latency_ms_bucket{direction="inbound", dst=~"(books.default.svc.cluster.local)(:\\d+)?"}[1m])) by (le, dst, rt_route))`,
						`sum(increase(route_response_total{direction="inbound", dst=~"(books.default.svc.cluster.local)(:\\d+)?"}[1m])) by (rt_route, dst, classification)`,
					},
					k8sConfigs: booksConfig,
				},
				req: pb.TopRoutesRequest{
					Selector: &pb.ResourceSelection{
						Resource: &pb.Resource{
							Namespace: "default",
							Type:      pkgK8s.Service,
							Name:      "books",
						},
					},
					TimeWindow: "1m",
					Outbound: &pb.TopRoutesRequest_None{
						None: &pb.Empty{},
					},
				},
				expectedResponse: GenTopRoutesResponse(routes, counts, "books"),
			},
		}

		testTopRoutes(t, expectations)
	})

	t.Run("Successfully performs an outbound routes query", func(t *testing.T) {
		routes := []string{"/a"}
		counts := []uint64{123}
		expectations := []topRoutesExpected{
			topRoutesExpected{
				expectedStatRPC: expectedStatRPC{
					err:              nil,
					mockPromResponse: routesMetric([]string{"/a"}),
					expectedPrometheusQueries: []string{
						`histogram_quantile(0.5, sum(irate(route_response_latency_ms_bucket{deployment="traffic", direction="outbound", dst=~"(books.default.svc.cluster.local)(:\\d+)?", namespace="default"}[1m])) by (le, dst, rt_route))`,
						`histogram_quantile(0.95, sum(irate(route_response_latency_ms_bucket{deployment="traffic", direction="outbound", dst=~"(books.default.svc.cluster.local)(:\\d+)?", namespace="default"}[1m])) by (le, dst, rt_route))`,
						`histogram_quantile(0.99, sum(irate(route_response_latency_ms_bucket{deployment="traffic", direction="outbound", dst=~"(books.default.svc.cluster.local)(:\\d+)?", namespace="default"}[1m])) by (le, dst, rt_route))`,
						`sum(increase(route_response_total{deployment="traffic", direction="outbound", dst=~"(books.default.svc.cluster.local)(:\\d+)?", namespace="default"}[1m])) by (rt_route, dst, classification)`,
					},
					k8sConfigs: booksConfig,
				},
				req: pb.TopRoutesRequest{
					Selector: &pb.ResourceSelection{
						Resource: &pb.Resource{
							Namespace: "default",
							Type:      pkgK8s.Deployment,
							Name:      "traffic",
						},
					},
					Outbound: &pb.TopRoutesRequest_ToResource{
						ToResource: &pb.Resource{
							Type: pkgK8s.Service,
						},
					},
					TimeWindow: "1m",
				},
				expectedResponse: GenTopRoutesResponse(routes, counts, "books"),
			},
		}

		testTopRoutes(t, expectations)
	})

	t.Run("Successfully performs an outbound authority query", func(t *testing.T) {
		routes := []string{"/a"}
		counts := []uint64{123}
		expectations := []topRoutesExpected{
			topRoutesExpected{
				expectedStatRPC: expectedStatRPC{
					err:              nil,
					mockPromResponse: routesMetric([]string{"/a"}),
					expectedPrometheusQueries: []string{
						`histogram_quantile(0.5, sum(irate(route_response_latency_ms_bucket{deployment="traffic", direction="outbound", dst=~"(books.default.svc.cluster.local)(:\\d+)?", namespace="default"}[1m])) by (le, dst, rt_route))`,
						`histogram_quantile(0.95, sum(irate(route_response_latency_ms_bucket{deployment="traffic", direction="outbound", dst=~"(books.default.svc.cluster.local)(:\\d+)?", namespace="default"}[1m])) by (le, dst, rt_route))`,
						`histogram_quantile(0.99, sum(irate(route_response_latency_ms_bucket{deployment="traffic", direction="outbound", dst=~"(books.default.svc.cluster.local)(:\\d+)?", namespace="default"}[1m])) by (le, dst, rt_route))`,
						`sum(increase(route_response_total{deployment="traffic", direction="outbound", dst=~"(books.default.svc.cluster.local)(:\\d+)?", namespace="default"}[1m])) by (rt_route, dst, classification)`,
					},
					k8sConfigs: booksConfig,
				},
				req: pb.TopRoutesRequest{
					Selector: &pb.ResourceSelection{
						Resource: &pb.Resource{
							Namespace: "default",
							Type:      pkgK8s.Deployment,
							Name:      "traffic",
						},
					},
					Outbound: &pb.TopRoutesRequest_ToResource{
						ToResource: &pb.Resource{
							Type: pkgK8s.Authority,
							Name: "books.default.svc.cluster.local",
						},
					},
					TimeWindow: "1m",
				},
				expectedResponse: GenTopRoutesResponse(routes, counts, "books.default.svc.cluster.local"),
			},
		}

		testTopRoutes(t, expectations)
	})
}
