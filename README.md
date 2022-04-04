[![Build](https://github.com/jidicula/da-vinci/actions/workflows/build.yml/badge.svg)](https://github.com/jidicula/da-vinci/actions/workflows/build.yml) [![Latest Release](https://github.com/jidicula/da-vinci/actions/workflows/release-draft.yml/badge.svg)](https://github.com/jidicula/da-vinci/actions/workflows/release-draft.yml) [![Go Report Card](https://goreportcard.com/badge/github.com/jidicula/da-vinci)](https://goreportcard.com/report/github.com/jidicula/da-vinci) [![Go Reference](https://pkg.go.dev/badge/github.com/jidicula/da-vinci.svg)](https://pkg.go.dev/github.com/jidicula/da-vinci)

# Reddit r/place automation 2022

Scripts an image to r/place.

# Requirements
Go (https://go.dev) or download a release binary directly at https://github.com/jidicla/da-vinci/releases .

PNG image for drawing.

# How to Get App Client ID and App Secret Key
Steps:

1. Visit https://www.reddit.com/prefs/apps
2. Click "create (another) app" button at very bottom
3. Select the "script" option and fill in the fields with anything

# Usage

```
mv example_config.json config.json
```

and fill in values.

Run the app:

```
da-vinci example.png
```
