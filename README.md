# snssharecount

フューチャー技術ブログのSNSカウント更新プログラム。

## Installation

```sh
go install github.com/ma91n/snssharecount/cmd/snssharecount@latest
go install github.com/ma91n/snssharecount/cmd/ga@latest
```

## Usage

future-architect/tech-blog 直下で実行する想定。

SNSカウント

```sh
set http_proxy=<proxy url>
set https_proxy=<proxy url>
set FB_TOKEN=<Facebook Access Token>

snssharecount > temp.json
mv temp.json sns_count_cache.json
```

Google Analytics

```sh
ga > ga_cache.json
```

