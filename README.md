# go-rss-wechat

Just a simple go application to generate RSS for wechat subscriptions.

This application doesn't fetch data from wechat directly but from <http://www.jintiankansha.me/>. Thanks for wechat's **shit** policy.

# How to run

```bash
go run main.go [port]
```

Than access `http://localhost:[port]/rss/[name]`

# Config file

The config file is named `seeds.json`. The example is:

```json
[
  {
    "name": "Vista看天下",
    "url": "http://www.jintiankansha.me/column/PnmImrUtYi",
    "source": "jtks"
  },
  {
    "name": "浪潮工作室",
    "url": "http://www.jintiankansha.me/column/yxeK3uVmkK",
    "source": "jtks"
  }
]
```

- `name`: Name for this subscription. Used in the URL.
- `url`: URL for the article list.
- `source`: Now only support `jtks`.

# Docker Image

## Build

Just use the Dockerfile to build.

## Pre-build Image

<https://cloud.docker.com/u/shell32/repository/docker/shell32/go-rss-wechat>

Run `docker run -d -p 127.0.0.1:8081:8081 -v `pwd`:`pwd` -w `pwd` shell32/go-rss-wechat:1.0.0` in the same directory with config file.