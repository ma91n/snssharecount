package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli"
	ga "google.golang.org/api/analyticsdata/v1beta"
	//"google.golang.org/api/analyticsreporting/v4"
	"google.golang.org/api/option"
	"log"
	"os"
)

type GoogleAnalyticsCache struct {
	Monthly []GoogleAnalyticsPV `json:"monthly"`
	Weekly  []GoogleAnalyticsPV `json:"weekly"`
	Yearly  []GoogleAnalyticsPV `json:"yearly"`
}

type GoogleAnalyticsPV struct {
	Path  string `json:"path"`
	Pv    string `json:"pv"`
	Title string `json:"title"`
}

func main() {

	app := &cli.App{
		Name:  "google analytics",
		Usage: "ga",
		Action: func(cCtx *cli.Context) error {
			ctx := context.Background()

			// https://pkg.go.dev/google.golang.org/api/analyticsreporting/v4
			ars, err := ga.NewService(ctx, option.WithScopes(ga.AnalyticsReadonlyScope))
			if err != nil {
				return fmt.Errorf("new google analytice service: %w", err)
			}

			pvWeekly, err := fetchGoogleAnalytics(ars, ctx, "7daysAgo", "today")
			if err != nil {
				return err
			}
			pvMonthly, err := fetchGoogleAnalytics(ars, ctx, "30daysAgo", "today")
			if err != nil {
				return err
			}
			pvYearly, err := fetchGoogleAnalytics(ars, ctx, "365daysAgo", "today")
			if err != nil {
				return err
			}

			output := GoogleAnalyticsCache{
				Weekly:  pvWeekly,
				Monthly: pvMonthly,
				Yearly:  pvYearly,
			}

			outputJSON, err := json.MarshalIndent(output, "", " ")
			if err != nil {
				return fmt.Errorf("marshal json: %w", err)
			}

			fmt.Println(string(outputJSON))

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}

func fetchGoogleAnalytics(ars *ga.Service, ctx context.Context, start, end string) ([]GoogleAnalyticsPV, error) {
	resp, err := ars.Properties.BatchRunReports("properties/319098367", &ga.BatchRunReportsRequest{
		Requests: []*ga.RunReportRequest{
			{
				Property:   "properties/319098367",
				DateRanges: []*ga.DateRange{{StartDate: start, EndDate: end}},
				Dimensions: []*ga.Dimension{{Name: "pagePath"}, {Name: "pageTitle"}},
				Metrics:    []*ga.Metric{{Name: "screenPageViews"}},
				OrderBys:   []*ga.OrderBy{{Desc: true, Dimension: &ga.DimensionOrderBy{DimensionName: "screenPageViews"}}},
				Limit:      50,
			},
		},
	}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("reporting batch get: %w", err)
	}

	pvs := make([]GoogleAnalyticsPV, 0, 50)
	for _, report := range resp.Reports {
		for _, row := range report.Rows {
			pvs = append(pvs, GoogleAnalyticsPV{
				Path:  row.DimensionValues[0].Value,
				Pv:    row.MetricValues[0].Value,
				Title: row.DimensionValues[1].Value,
			})
		}
	}
	return pvs, nil
}
