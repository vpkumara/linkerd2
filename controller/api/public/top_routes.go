package public

import (
	"context"
	"fmt"
	"sort"
	"strings"

	sp "github.com/linkerd/linkerd2/controller/gen/apis/serviceprofile/v1alpha1"
	pb "github.com/linkerd/linkerd2/controller/gen/public"
	"github.com/linkerd/linkerd2/pkg/k8s"
	"github.com/prometheus/common/model"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	routeReqQuery             = "sum(increase(route_response_total%s[%s])) by (%s, dst, classification)"
	routeLatencyQuantileQuery = "histogram_quantile(%s, sum(irate(route_response_latency_ms_bucket%s[%s])) by (le, dst, %s))"
	dstLabel                  = `dst=~"(%s)(:\\d+)?"`
	DefaultRouteName          = "[DEFAULT]"
)

type dstAndRoute struct {
	dst   string
	route string
}

type indexedTable = map[dstAndRoute]*pb.RouteTable_Row

func (s *grpcServer) TopRoutes(ctx context.Context, req *pb.TopRoutesRequest) (*pb.TopRoutesResponse, error) {

	// check for well-formed request
	if req.GetSelector().GetResource() == nil {
		return topRoutesError(req, "TopRoutes request missing Selector Resource"), nil
	}

	if req.GetNone() == nil {
		// This is an outbound (--to) request.
		targetType := req.GetSelector().GetResource().GetType()
		if targetType == k8s.Service || targetType == k8s.Authority {
			return nil, fmt.Errorf("The %s resource type is not supported with 'to' queries", targetType)
		}
	}

	table := make(indexedTable)

	if req.GetToAll() != nil {
		// Deprecated, use to ToResource instead.
		ps, err := s.k8sAPI.SP().Lister().ServiceProfiles(s.controllerNamespace).List(labels.Everything())
		if err != nil {
			return nil, err
		}

		if len(ps) == 0 {
			return topRoutesError(req, "No ServiceProfiles found"), nil
		}

		for _, p := range ps {
			addRouteRows(table, p, p.GetName(), p.GetName())
		}
	} else if dst := req.GetToAuthority(); dst != "" {
		// Deprecated, use to ToResource instead.
		p, err := s.k8sAPI.SP().Lister().ServiceProfiles(s.controllerNamespace).Get(dst)
		if err != nil {
			return topRoutesError(req, fmt.Sprintf("No ServiceProfile found for %s", dst)), nil
		}
		addRouteRows(table, p, dst, dst)
	} else {
		requestedResource := req.GetSelector().GetResource()
		if req.GetToResource() != nil {
			requestedResource = req.GetToResource()
		}

		if requestedResource.GetType() == k8s.Authority {
			if requestedResource.GetName() == "" {
				ps, err := s.k8sAPI.SP().Lister().ServiceProfiles(s.controllerNamespace).List(labels.Everything())
				if err != nil {
					return nil, err
				}

				if len(ps) == 0 {
					return topRoutesError(req, "No ServiceProfiles found"), nil
				}

				for _, p := range ps {
					addRouteRows(table, p, p.GetName(), p.GetName())
				}
			} else {
				p, err := s.k8sAPI.SP().Lister().ServiceProfiles(s.controllerNamespace).Get(requestedResource.GetName())
				if err != nil {
					return nil, err
				}
				addRouteRows(table, p, p.GetName(), p.GetName())
			}
		} else {
			objects, err := s.k8sAPI.GetObjects(requestedResource.Namespace, requestedResource.Type, requestedResource.Name)
			if err != nil {
				return nil, err
			}

			for _, obj := range objects {
				services, err := s.k8sAPI.GetServicesFor(obj, false)
				if err != nil {
					return nil, err
				}
				for _, svc := range services {
					dst := fmt.Sprintf("%s.%s.svc.cluster.local", svc.Name, svc.Namespace)
					p, err := s.k8sAPI.SP().Lister().ServiceProfiles(s.controllerNamespace).Get(dst)
					if err != nil {
						// No ServiceProfile found for this Service.  Move on.
						continue
					}
					addRouteRows(table, p, dst, svc.GetName())
				}
			}

			if len(table) == 0 {
				return topRoutesError(req, fmt.Sprintf("No ServiceProfiles found for Services of %s/%s", requestedResource.Type, requestedResource.Name)), nil
			}
		}
	}

	err := s.getRouteMetrics(ctx, req, table)
	if err != nil {
		return nil, err
	}

	rows := make([]*pb.RouteTable_Row, 0)
	for _, row := range table {
		rows = append(rows, row)
	}

	return &pb.TopRoutesResponse{
		Response: &pb.TopRoutesResponse_Routes{
			Routes: &pb.RouteTable{
				Rows: rows,
			},
		},
	}, nil
}

func addRouteRows(table indexedTable, profile *sp.ServiceProfile, dst string, authority string) {
	for _, route := range profile.Spec.Routes {
		key := dstAndRoute{
			dst,
			route.Name,
		}
		table[key] = &pb.RouteTable_Row{
			Authority: authority,
			Route:     route.Name,
			Stats:     &pb.BasicStats{},
		}
	}
	defaultRoute := dstAndRoute{
		dst,
		"",
	}
	table[defaultRoute] = &pb.RouteTable_Row{
		Authority: authority,
		Route:     DefaultRouteName,
		Stats:     &pb.BasicStats{},
	}
}

func topRoutesError(req *pb.TopRoutesRequest, message string) *pb.TopRoutesResponse {
	return &pb.TopRoutesResponse{
		Response: &pb.TopRoutesResponse_Error{
			Error: &pb.ResourceError{
				Resource: req.GetSelector().GetResource(),
				Error:    message,
			},
		},
	}
}

func (s *grpcServer) getRouteMetrics(ctx context.Context, req *pb.TopRoutesRequest, table indexedTable) error {
	timeWindow := req.TimeWindow
	reqLabels, err := s.buildRouteLabels(req)
	if err != nil {
		return err
	}
	groupBy := "rt_route"

	results, err := s.getPrometheusMetrics(ctx, routeReqQuery, routeLatencyQuantileQuery, reqLabels, timeWindow, groupBy)
	if err != nil {
		return err
	}

	processRouteMetrics(results, timeWindow, table)
	return nil
}

func (s *grpcServer) buildRouteLabels(req *pb.TopRoutesRequest) (string, error) {
	// labels: the labels for the resource we want to query for
	var labels model.LabelSet

	switch out := req.Outbound.(type) {

	case *pb.TopRoutesRequest_ToAuthority:
		// Deprecated.  Use ToResource.
		labels = labels.Merge(promQueryLabels(req.Selector.Resource))
		labels = labels.Merge(promDirectionLabels("outbound"))
		return renderLabels(labels, []string{out.ToAuthority}), nil

	case *pb.TopRoutesRequest_ToAll:
		// Deprecated.  Use ToResource.
		labels = labels.Merge(promQueryLabels(req.Selector.Resource))
		labels = labels.Merge(promDirectionLabels("outbound"))
		return renderLabels(labels, []string{}), nil

	case *pb.TopRoutesRequest_ToResource:
		labels = labels.Merge(promQueryLabels(req.Selector.Resource))
		labels = labels.Merge(promDirectionLabels("outbound"))

		serviceNames := make([]string, 0)
		if out.ToResource.GetType() == k8s.Authority {
			if out.ToResource.GetName() != "" {
				serviceNames = []string{out.ToResource.GetName()}
			}
		} else {
			objects, err := s.k8sAPI.GetObjects(out.ToResource.Namespace, out.ToResource.Type, out.ToResource.Name)
			if err != nil {
				return "", err
			}

			for _, obj := range objects {
				services, err := s.k8sAPI.GetServicesFor(obj, false)
				if err != nil {
					return "", err
				}
				for _, svc := range services {
					serviceNames = append(serviceNames, fmt.Sprintf("%s.%s.svc.cluster.local", svc.Name, svc.Namespace))
				}
			}
		}
		return renderLabels(labels, serviceNames), nil

	default:
		labels = labels.Merge(promDirectionLabels("inbound"))

		if req.Selector.Resource.GetType() == k8s.Service {
			service := fmt.Sprintf("%s.%s.svc.cluster.local", req.Selector.Resource.GetName(), req.Selector.Resource.GetNamespace())
			return renderLabels(labels, []string{service}), nil
		}

		labels = labels.Merge(promQueryLabels(req.Selector.Resource))
		return renderLabels(labels, []string{}), nil
	}
}

func renderLabels(labels model.LabelSet, services []string) string {
	pairs := make([]string, 0)
	for k, v := range labels {
		pairs = append(pairs, fmt.Sprintf("%s=%q", k, v))
	}
	if len(services) > 0 {
		pairs = append(pairs, fmt.Sprintf(dstLabel, strings.Join(services, "|")))
	}
	sort.Strings(pairs)
	return fmt.Sprintf("{%s}", strings.Join(pairs, ", "))
}

func processRouteMetrics(results []promResult, timeWindow string, table indexedTable) {
	for _, result := range results {
		for _, sample := range result.vec {

			route := string(sample.Metric[model.LabelName("rt_route")])
			dst := string(sample.Metric[model.LabelName("dst")])
			dst = strings.Split(dst, ":")[0] // Truncate port, if there is one.

			key := dstAndRoute{dst, route}

			if table[key] == nil {
				log.Warnf("Found stats for unknown route: %s:%s", dst, route)
				continue
			}

			table[key].TimeWindow = timeWindow
			value := extractSampleValue(sample)

			switch result.prom {
			case promRequests:
				switch string(sample.Metric[model.LabelName("classification")]) {
				case "success":
					table[key].Stats.SuccessCount += value
				case "failure":
					table[key].Stats.FailureCount += value
				}
			case promLatencyP50:
				table[key].Stats.LatencyMsP50 = value
			case promLatencyP95:
				table[key].Stats.LatencyMsP95 = value
			case promLatencyP99:
				table[key].Stats.LatencyMsP99 = value
			}
		}
	}
}
