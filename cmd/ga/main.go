package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli"
	"google.golang.org/api/analyticsreporting/v4"
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
		Name:  "sns share count",
		Usage: "snssharecount <URL>",
		Action: func(cCtx *cli.Context) error {
			ctx := context.Background()

			// https://pkg.go.dev/google.golang.org/api/analyticsreporting/v4
			ars, err := analyticsreporting.NewService(ctx, option.WithScopes(analyticsreporting.AnalyticsReadonlyScope))
			if err != nil {
				return fmt.Errorf("new google analytice service: %w", err)
			}

			pvWeekly, err := fetchGoogleAnalytics(ars, ctx, "7daysAgo", "1daysAgo")
			if err != nil {
				return err
			}
			pvMonthly, err := fetchGoogleAnalytics(ars, ctx, "30daysAgo", "1daysAgo")
			if err != nil {
				return err
			}
			pvYearly, err := fetchGoogleAnalytics(ars, ctx, "365daysAgo", "1daysAgo")
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

func fetchGoogleAnalytics(ars *analyticsreporting.Service, ctx context.Context, start, end string) ([]GoogleAnalyticsPV, error) {
	resp, err := ars.Reports.BatchGet(&analyticsreporting.GetReportsRequest{
		ReportRequests: []*analyticsreporting.ReportRequest{{
			DateRanges: []*analyticsreporting.DateRange{
				{StartDate: start, EndDate: end}},
			Dimensions: []*analyticsreporting.Dimension{
				{Name: "ga:pagePath"},
				{Name: "ga:pageTitle"},
			},
			Metrics: []*analyticsreporting.Metric{
				{Expression: "ga:pageviews"},
			},
			OrderBys: []*analyticsreporting.OrderBy{
				{
					FieldName: "ga:pageviews",
					SortOrder: "DESCENDING",
				},
			},
			PageSize: 50,
			ViewId:   "117039269",
		}},
	}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("reporting batch get: %w", err)
	}

	pvs := make([]GoogleAnalyticsPV, 0, 50)
	for _, report := range resp.Reports {
		for _, row := range report.Data.Rows {
			pvs = append(pvs, GoogleAnalyticsPV{
				Path:  row.Dimensions[0],
				Pv:    row.Metrics[0].Values[0],
				Title: row.Dimensions[1],
			})
		}
	}
	return pvs, nil
}
