package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli"
	"golang.org/x/exp/maps"
	ga "google.golang.org/api/analyticsdata/v1beta"
	"math"
	"strconv"
	"strings"

	//"google.golang.org/api/analyticsreporting/v4"
	"google.golang.org/api/option"
	"log"
	"os"
)

type GoogleAnalyticsCache struct {
	PV []GoogleAnalyticsPV `json:"pv"`
}

type GoogleAnalyticsPV struct {
	Path  string `json:"path"`
	Pv    int    `json:"pv"`
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

			pvYearly, err := fetchGoogleAnalytics(ars, ctx, "2016-02-16", "today") // フューチャー技術ブログの創立日から
			if err != nil {
				return err
			}

			output := GoogleAnalyticsCache{
				PV: pvYearly,
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
				Limit:      3000,
			},
		},
	}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("reporting batch get: %w", err)
	}

	pvsMap := make(map[string]GoogleAnalyticsPV, 1000)
	pathEmptyPVMap := make(map[string]int, 1000)
	for _, report := range resp.Reports {
		for _, row := range report.Rows {

			viewCnt, err := strconv.Atoi(row.MetricValues[0].Value)
			if err != nil {
				viewCnt = 0
			}
			roundupPV := int(math.Ceil(float64(viewCnt)/100)) * 100

			urlPath := row.DimensionValues[0].Value
			articleTitle := row.DimensionValues[1].Value

			if len(urlPath) == 0 {
				// 原因が不明だが空っぽのケースがある
				pathEmptyPVMap[articleTitle] = roundupPV
				continue
			}

			if strings.HasPrefix(urlPath, "/tags/") || strings.HasPrefix(urlPath, "/authors/") ||
				strings.HasPrefix(urlPath, "/categories/") {
				continue
			}

			if strings.HasSuffix(urlPath, "index.html") {
				// URLに index.html が含まれるケースがある
				urlPath = strings.ReplaceAll(urlPath, "index.html", "")
			}

			analyticsPV, ok := pvsMap[articleTitle]
			if ok {
				analyticsPV.Pv += roundupPV
				pvsMap[articleTitle] = analyticsPV
				continue
			}

			pvsMap[articleTitle] = GoogleAnalyticsPV{
				Path:  urlPath,
				Pv:    roundupPV,
				Title: articleTitle,
			}
		}
	}

	// タイトルで当ててPV数を合算
	for k, v := range pathEmptyPVMap {
		analyticsPV, ok := pvsMap[k]
		if !ok {
			// URL不明のため諦める
			continue
		}
		analyticsPV.Pv += v
		pvsMap[k] = analyticsPV
	}

	// URLで一意にする（途中でタイトル変更があると揺れるため）
	uniquePVs := make(map[string]GoogleAnalyticsPV, len(pvsMap))
	for _, v := range pvsMap {
		analyticsPV, ok := uniquePVs[v.Path]
		if !ok {
			uniquePVs[v.Path] = v
			continue
		}
		analyticsPV.Pv += v.Pv
		uniquePVs[v.Path] = analyticsPV
	}

	return maps.Values(uniquePVs), nil
}
