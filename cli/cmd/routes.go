package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/linkerd/linkerd2/controller/api/util"
	pb "github.com/linkerd/linkerd2/controller/gen/public"
	"github.com/linkerd/linkerd2/pkg/k8s"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type routesOptions struct {
	statOptionsBase
	toResource   string
	toNamespace  string
	dstIsService bool
}

func newRoutesOptions() *routesOptions {
	return &routesOptions{
		statOptionsBase: *newStatOptionsBase(),
		toResource:      "",
		toNamespace:     "",
	}
}

func newCmdRoutes() *cobra.Command {
	options := newRoutesOptions()

	cmd := &cobra.Command{
		Use:   "routes [flags] (RESOURCES)",
		Short: "Display route stats",
		Long: `Display route stats.

This command will only display traffic which is sent to a service that has a Service Profile defined.`,
		Example: `  # Routes for the webapp service in the test namespace.
  linkerd routes service/webapp -n test

  # Routes for calls from from the traffic deployment to the webapp service in the test namespace.
  linkerd routes deploy/traffic -n test --to svc/webapp`,
		Args:      cobra.ExactArgs(1),
		ValidArgs: util.ValidTargets,
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := buildTopRoutesRequest(args[0], options)
			if err != nil {
				return fmt.Errorf("error creating metrics request while making routes request: %v", err)
			}

			output, err := requestRouteStatsFromAPI(validatedPublicAPIClient(time.Time{}), req, options)
			if err != nil {
				return err
			}

			_, err = fmt.Print(output)

			return err
		},
	}

	cmd.PersistentFlags().StringVarP(&options.namespace, "namespace", "n", options.namespace, "Namespace of the specified resource")
	cmd.PersistentFlags().StringVarP(&options.timeWindow, "time-window", "t", options.timeWindow, "Stat window (for example: \"10s\", \"1m\", \"10m\", \"1h\")")
	cmd.PersistentFlags().StringVar(&options.toResource, "to", options.toResource, "If present, shows outbound stats to the specified resource")
	cmd.PersistentFlags().StringVar(&options.toNamespace, "to-namespace", options.toNamespace, "Sets the namespace used to lookup the \"--to\" resource; by default the current \"--namespace\" is used")
	cmd.PersistentFlags().StringVarP(&options.outputFormat, "output", "o", options.outputFormat, "Output format; currently only \"table\" (default) and \"json\" are supported")

	return cmd
}

func requestRouteStatsFromAPI(client pb.ApiClient, req *pb.TopRoutesRequest, options *routesOptions) (string, error) {
	resp, err := client.TopRoutes(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("TopRoutes API error: %v", err)
	}
	if e := resp.GetError(); e != nil {
		return "", errors.New(e.Error)
	}

	return renderRouteStats(resp, options), nil
}

func renderRouteStats(resp *pb.TopRoutesResponse, options *routesOptions) string {
	var buffer bytes.Buffer
	w := tabwriter.NewWriter(&buffer, 0, 0, padding, ' ', tabwriter.AlignRight)
	writeRouteStatsToBuffer(resp, w, options)
	w.Flush()

	return renderStats(buffer, &options.statOptionsBase)
}

func writeRouteStatsToBuffer(resp *pb.TopRoutesResponse, w *tabwriter.Writer, options *routesOptions) {
	table := make([]*rowStats, 0)

	for _, r := range resp.GetRoutes().Rows {
		if r.Stats != nil {
			route := r.GetRoute()
			table = append(table, &rowStats{
				route:       route,
				dst:         r.GetAuthority(),
				requestRate: getRequestRate(r.Stats, r.TimeWindow),
				successRate: getSuccessRate(r.Stats),
				latencyP50:  r.Stats.LatencyMsP50,
				latencyP95:  r.Stats.LatencyMsP95,
				latencyP99:  r.Stats.LatencyMsP99,
			})
		}
	}

	sort.Slice(table, func(i, j int) bool {
		return table[i].dst+table[i].route < table[j].dst+table[j].route
	})

	switch options.outputFormat {
	case "table", "":
		if len(table) == 0 {
			fmt.Fprintln(os.Stderr, "No traffic found.  Does the service have a service profile?  You can create one with the `linkerd profile` command.")
			os.Exit(0)
		}
		printRouteTable(table, w, options)
	case "json":
		printRouteJSON(table, w)
	}
}

func printRouteTable(stats []*rowStats, w *tabwriter.Writer, options *routesOptions) {
	// template for left-aligning the route column
	routeTemplate := fmt.Sprintf("%%-%ds", routeWidth(stats))

	authorityColumn := "AUTHORITY"
	if options.dstIsService {
		authorityColumn = "SERVICE"
	}

	headers := []string{
		fmt.Sprintf(routeTemplate, "ROUTE"),
		authorityColumn,
		"SUCCESS",
		"RPS",
		"LATENCY_P50",
		"LATENCY_P95",
		"LATENCY_P99\t", // trailing \t is required to format last column
	}

	fmt.Fprintln(w, strings.Join(headers, "\t"))

	templateString := routeTemplate + "\t%s\t%.2f%%\t%.1frps\t%dms\t%dms\t%dms\t\n"

	for _, row := range stats {
		fmt.Fprintf(w, templateString,
			row.route,
			row.dst,
			row.successRate*100,
			row.requestRate,
			row.latencyP50,
			row.latencyP95,
			row.latencyP99,
		)
	}
}

// Using pointers there where the value is NA and the corresponding json is null
type jsonRouteStats struct {
	Route        string   `json:"route"`
	Authority    string   `json:"authority"`
	Success      *float64 `json:"success"`
	Rps          *float64 `json:"rps"`
	LatencyMSp50 *uint64  `json:"latency_ms_p50"`
	LatencyMSp95 *uint64  `json:"latency_ms_p95"`
	LatencyMSp99 *uint64  `json:"latency_ms_p99"`
}

func printRouteJSON(stats []*rowStats, w *tabwriter.Writer) {
	// avoid nil initialization so that if there are not stats it gets marshalled as an empty array vs null
	entries := []*jsonRouteStats{}
	for _, row := range stats {
		route := row.route
		entry := &jsonRouteStats{
			Route: route,
		}

		entry.Authority = row.dst
		entry.Success = &row.successRate
		entry.Rps = &row.requestRate
		entry.LatencyMSp50 = &row.latencyP50
		entry.LatencyMSp95 = &row.latencyP95
		entry.LatencyMSp99 = &row.latencyP99

		entries = append(entries, entry)
	}
	b, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		log.Error(err.Error())
		return
	}
	fmt.Fprintf(w, "%s\n", b)
}

func buildTopRoutesRequest(resource string, options *routesOptions) (*pb.TopRoutesRequest, error) {
	err := options.validateOutputFormat()
	if err != nil {
		return nil, err
	}

	target, err := util.BuildResource(options.namespace, resource)
	if err != nil {
		return nil, err
	}

	requestParams := util.TopRoutesRequestParams{
		StatsBaseRequestParams: util.StatsBaseRequestParams{
			TimeWindow:   options.timeWindow,
			ResourceName: target.Name,
			ResourceType: target.Type,
			Namespace:    options.namespace,
		},
	}

	options.dstIsService = !(target.GetType() == k8s.Authority)

	if options.toResource != "" {
		if options.toNamespace == "" {
			options.toNamespace = options.namespace
		}
		toRes, err := util.BuildResource(options.toNamespace, options.toResource)
		if err != nil {
			return nil, err
		}

		options.dstIsService = !(toRes.GetType() == k8s.Authority)

		requestParams.ToName = toRes.Name
		requestParams.ToNamespace = toRes.Namespace
		requestParams.ToType = toRes.Type
	}

	return util.BuildTopRoutesRequest(requestParams)
}

// returns the length of the longest route name
func routeWidth(stats []*rowStats) int {
	maxLength := 0
	for _, row := range stats {
		if len(row.route) > maxLength {
			maxLength = len(row.route)
		}
	}
	return maxLength
}
