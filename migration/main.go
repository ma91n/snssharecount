package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"time"
)

type URLShareCnt struct {
	URL      string
	Twitter  ShareCnt
	FaceBook ShareCnt
	Hatebu   ShareCnt
	Pocket   ShareCnt
}

type ShareCnt struct {
	Count   int
	FetchAt time.Time
}

func main() {
	fbcache, err := os.Open("resource/cache_facebook.json")
	if err != nil {
		panic(err)
	}
	fbAll, err := io.ReadAll(fbcache)
	if err != nil {
		panic(err)
	}
	var fbmap map[string]int
	if err := json.Unmarshal(fbAll, &fbmap); err != nil {
		panic(err)
	}

	hatebucache, err := os.Open("resource/cache_hatebu.json")
	if err != nil {
		panic(err)
	}
	hatebuAll, err := io.ReadAll(hatebucache)
	if err != nil {
		panic(err)
	}
	var hatebumap map[string]int
	if err := json.Unmarshal(hatebuAll, &hatebumap); err != nil {
		panic(err)
	}

	pocketcache, err := os.Open("resource/cache_pocket.json")
	if err != nil {
		panic(err)
	}
	pocketAll, err := io.ReadAll(pocketcache)
	if err != nil {
		panic(err)
	}
	var pocketmap map[string]int
	if err := json.Unmarshal(pocketAll, &pocketmap); err != nil {
		panic(err)
	}

	twcache, err := os.Open("resource/cache_twitter.json")
	if err != nil {
		panic(err)
	}
	twAll, err := io.ReadAll(twcache)
	if err != nil {
		panic(err)
	}
	var twmap map[string]int
	if err := json.Unmarshal(twAll, &twmap); err != nil {
		panic(err)
	}

	now := time.Now()

	var shares []URLShareCnt
	for k, _ := range pocketmap {
		shares = append(shares, URLShareCnt{
			URL: k,
			Twitter: ShareCnt{
				Count:   twmap[k],
				FetchAt: now,
			},
			FaceBook: ShareCnt{
				Count:   fbmap[k],
				FetchAt: now,
			},
			Hatebu: ShareCnt{
				Count:   hatebumap[k],
				FetchAt: now,
			},
			Pocket: ShareCnt{
				Count:   pocketmap[k],
				FetchAt: now,
			},
		})
	}

	sort.SliceStable(shares, func(i, j int) bool {
		return shares[i].URL > shares[j].URL
	})

	marshal, err := json.MarshalIndent(shares, "", " ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(marshal))

}
