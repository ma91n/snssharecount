package main

import (
	"encoding/json"
	"fmt"
	"github.com/urfave/cli"
	"golang.org/x/exp/slices"
	"sort"
	"strings"

	"github.com/snabb/sitemap"
	"golang.org/x/exp/maps"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type URLShareCnt struct {
	URL      string
	Twitter  ShareCnt
	FaceBook ShareCnt
	Hatebu   ShareCnt
	Pocket   ShareCnt
	Feedly   *ShareCnt `json:",omitempty"`
}

type ShareCnt struct {
	Count   int
	FetchAt time.Time
}

type PocketResponse struct {
	Saves int `json:"saves"`
}

type FaceBookResponse struct {
	OgObject *OgObject
	ID       string `json:"id"`
}

func (c FaceBookResponse) Likes() int {
	if c.OgObject == nil {
		return 0
	}
	if c.OgObject.Engagement == nil {
		return 0
	}
	return c.OgObject.Engagement.Count
}

type OgObject struct {
	Engagement *Engagement `json:"engagement"`
	ID         string      `json:"id"`
}
type Engagement struct {
	Count          int    `json:"count"`
	SocialSentence string `json:"social_sentence"`
}

type FeedlyResponse struct {
	Description         string   `json:"description"`
	Language            string   `json:"language"`
	ID                  string   `json:"id"`
	Title               string   `json:"title"`
	FeedID              string   `json:"feedId"`
	Website             string   `json:"website"`
	Topics              []string `json:"topics"`
	Subscribers         int      `json:"subscribers"`
	Velocity            float64  `json:"velocity"`
	Updated             int64    `json:"updated"`
	IconURL             string   `json:"iconUrl"`
	Partial             bool     `json:"partial"`
	CoverURL            string   `json:"coverUrl"`
	VisualURL           string   `json:"visualUrl"`
	EstimatedEngagement int      `json:"estimatedEngagement"`
}

func main() {

	app := &cli.App{
		Name:  "sns share count",
		Usage: "snssharecount <URL>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "disable",
				Value: "",
				Usage: "disable refresh sns count. eg) disable=facebook,hatebu,pocket",
			},
			&cli.IntFlag{
				Name:  "days",
				Value: 14,
				Usage: "enable to reset sns count before days of this flag",
			},
		},
		Action: func(cCtx *cli.Context) error {

			snsCacheMap, err := readSNSCacheJSON()
			if err != nil {
				return err
			}

			postSitemap, err := fetchSiteMap()
			if err != nil {
				return err
			}

			// 公開後14日は更新する
			now := time.Now()
			resetMinute := now.Add(-1 * time.Minute)
			cacheResetAt := now.AddDate(0, 0, -1*cCtx.Int("days"))

			disables := strings.Split(cCtx.String("disable"), ",")

			defaultCnt := ShareCnt{
				Count: 0,
			}

			for _, v := range postSitemap.URLs {

				if v.Loc == "https://future-architect.github.io/" {
					feedly, err := fetchFeedly()
					if err != nil {
						return err
					}
					cache := snsCacheMap[v.Loc]
					cache.Feedly = &ShareCnt{
						Count:   feedly.Subscribers,
						FetchAt: now,
					}
					snsCacheMap[v.Loc] = cache
				}

				if v.LastMod != nil && v.LastMod.After(cacheResetAt) {
					// 記事の更新日がリセットより後ろであればSNSシェア数を更新

					cache, ok := snsCacheMap[v.Loc]
					if !ok {
						cache = URLShareCnt{
							URL:      v.Loc,
							Twitter:  defaultCnt,
							FaceBook: defaultCnt,
							Hatebu:   defaultCnt,
							Pocket:   defaultCnt,
						}
					}

					if cache.Pocket.FetchAt.Before(resetMinute) && !slices.Contains(disables, "pocket") {
						pc, err := fetchPocket(v.Loc)
						if err != nil {
							return err
						}
						cache.Pocket = ShareCnt{
							Count:   pc.Saves,
							FetchAt: now,
						}
					}

					if cache.Hatebu.FetchAt.Before(resetMinute) && !slices.Contains(disables, "hatebu") {
						hatebuCnt, err := fetchHatebu(v.Loc)
						if err != nil {
							return err
						}
						cache.Hatebu = ShareCnt{
							Count:   hatebuCnt,
							FetchAt: now,
						}
					}

					if cache.FaceBook.FetchAt.Before(resetMinute) && !slices.Contains(disables, "facebook") {
						fbCnt, err := fetchFacebook(v.Loc)
						if err != nil {
							return err
						}
						cache.FaceBook = ShareCnt{
							Count:   fbCnt.Likes(),
							FetchAt: now,
						}
					}

					snsCacheMap[v.Loc] = cache
				}
			}

			shares := maps.Values(snsCacheMap)
			sort.SliceStable(shares, func(i, j int) bool {
				return shares[i].URL > shares[j].URL
			})

			marshal, err := json.MarshalIndent(shares, "", " ")
			if err != nil {
				panic(err)
			}
			fmt.Println(string(marshal))

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}

}

func fetchSiteMap() (*sitemap.Sitemap, error) {
	resp, err := http.Get("https://future-architect.github.io/post-sitemap.xml")
	if err != nil {
		return nil, fmt.Errorf("get post-sitemap.xml: %w", err)
	}
	defer resp.Body.Close()

	sm := sitemap.New()
	if _, err := sm.ReadFrom(resp.Body); err != nil {
		return nil, fmt.Errorf("parse post-sitemap.xml: %w", err)
	}
	return sm, nil
}

func readSNSCacheJSON() (map[string]URLShareCnt, error) {
	snsCache, err := os.Open("sns_count_cache.json")
	if err != nil {
		return nil, fmt.Errorf("sns_count_cache.json is not found: %w", err)
	}
	defer snsCache.Close()

	all, err := io.ReadAll(snsCache)
	if err != nil {
		return nil, fmt.Errorf("read sns_count_cache.json: %w", err)
	}

	var shares []URLShareCnt
	if err := json.Unmarshal(all, &shares); err != nil {
		return nil, fmt.Errorf("unmarshal sns_count_cache.json: %w", err)
	}

	shareMap := make(map[string]URLShareCnt, len(shares))
	for _, v := range shares {
		shareMap[v.URL] = v
	}
	return shareMap, nil
}

func fetchFeedly() (FeedlyResponse, error) {
	resp, err := http.Get("https://cloud.feedly.com/v3/feeds/feed%2Fhttps%3A%2F%2Ffuture-architect.github.io%2Fatom.xml")
	if err != nil {
		return FeedlyResponse{}, fmt.Errorf("feedly: %w", err)
	}
	defer resp.Body.Close()

	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return FeedlyResponse{}, fmt.Errorf("read feedly response: %w", err)
	}

	var feedly FeedlyResponse
	if err := json.Unmarshal(all, &feedly); err != nil {
		return FeedlyResponse{}, fmt.Errorf("unmarshal feedly response: %w", err)
	}
	return feedly, nil
}

func fetchPocket(url string) (PocketResponse, error) {
	req, err := http.NewRequest(http.MethodGet, "https://widgets.getpocket.com/api/saves", nil)
	if err != nil {
		return PocketResponse{}, fmt.Errorf("pocket request url: %w", err)
	}
	q := req.URL.Query()
	q.Add("url", url)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return PocketResponse{}, fmt.Errorf("pocket count: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return PocketResponse{}, fmt.Errorf("pocket response body: %w", err)
	}

	var pc PocketResponse
	if err := json.Unmarshal(body, &pc); err != nil {
		return PocketResponse{}, fmt.Errorf("pocket response unmarshal json: %w", err)
	}
	return pc, nil
}

func fetchHatebu(url string) (int, error) {
	req, err := http.NewRequest(http.MethodGet, "https://bookmark.hatenaapis.com/count/entry", nil)
	if err != nil {
		return 0, fmt.Errorf("hatebu request url: %w", err)
	}
	q := req.URL.Query()
	q.Add("url", url)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("hatebu count: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("hatebu response body: %w", err)
	}

	bookMarkCnt, err := strconv.Atoi(string(body))
	if err != nil {
		return 0, fmt.Errorf("hatebu response is invalid: %w", err)
	}
	return bookMarkCnt, nil
}

func fetchFacebook(url string) (FaceBookResponse, error) {
	req, err := http.NewRequest(http.MethodGet, "https://graph.facebook.com/v10.0/?fields=og_object{engagement}", nil)
	if err != nil {
		return FaceBookResponse{}, fmt.Errorf("facebook request url: %w", err)
	}
	q := req.URL.Query()
	q.Add("id", url)
	q.Add("access_token", os.Getenv("FB_TOKEN"))
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return FaceBookResponse{}, fmt.Errorf("facebook http get count: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return FaceBookResponse{}, fmt.Errorf("facebook response body: %w", err)
	}

	resBody := string(body)
	fmt.Println(resBody)

	var fbc FaceBookResponse
	if err := json.Unmarshal(body, &fbc); err != nil {
		return FaceBookResponse{}, fmt.Errorf("facebook response unmarshal json: %w", err)
	}
	return fbc, nil
}
