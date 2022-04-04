package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/andrewstuart/rplace"
	"github.com/gopuff/morecontext"
)

type Config struct {
	Cli          Client    `json:"client"`
	Accounts     []Account `json:"accounts"`
	X            int       `json:"X"`
	Y            int       `json:"Y"`
	SleepSeconds int       `json:"SleepSeconds"`
}

type Client struct {
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
}

type TokenResp struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

var colorMap = map[string]int{
	"burgundy":    0,
	"dark red":    1,
	"red":         2,
	"orange":      3,
	"yellow":      4,
	"pale yellow": 5,
	"dark green":  6,
	"green":       7,
	"light green": 8,
	"dark teal":   9,
	"teal":        10,
	"light teal":  11,
	"dark blue":   12,
	"blue":        13,
	"light blue":  14,
	"indigo":      15,
	"periwinkle":  16,
	"lavender":    17,
	"dark purple": 18,
	"purple":      19,
	"pale purple": 20,
	"magenta":     21,
	"pink":        22,
	"light pink":  23,
	"dark brown":  24,
	"brown":       25,
	"beige":       26,
	"black":       27,
	"dark gray":   28,
	"gray":        29,
	"light gray":  30,
	"white":       31,
}

type Account struct {
	Username          string `json:"username"`
	Password          string `json:"password"`
	nextAvailableTime time.Time
	tokenExpiry       time.Time
	token             string
}

// getAuthToken adds a new auth token string and its expiry to the account.
func (a *Account) getAuthToken(c Client) error {
	client := &http.Client{}
	data := fmt.Sprintf(`grant_type=password&username=%s&password=%s`, url.QueryEscape(a.Username), url.QueryEscape(a.Password))

	body := bytes.NewBuffer([]byte(data))

	req, err := http.NewRequest("POST", "https://ssl.reddit.com/api/v1/access_token", body)
	if err != nil {
		return err
	}

	clientString := fmt.Sprintf("%s:%s", c.ClientID, c.ClientSecret)
	encodedClient := base64.StdEncoding.EncodeToString([]byte(clientString))
	basicAuth := fmt.Sprintf("Basic %s", encodedClient)

	req.Header.Add("Authorization", basicAuth)
	req.Header.Add("User-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.4 Safari/605.1.15")

	currentTime := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	t := TokenResp{}
	err = json.NewDecoder(resp.Body).Decode(&t)
	if err != nil {
		return err
	}

	a.token = t.AccessToken
	a.tokenExpiry = currentTime.Add(time.Second * time.Duration(t.ExpiresIn))
	return nil
}

// setPixel writes colorIndex to point p using token t.
func (a *Account) setPixel(p image.Point, colorIdx int) error {

	canvasIdx := 0
	// Using for instead of if in case canvas expands again.
	for p.X > 999 {
		p.X -= 1000
		canvasIdx++
	}
	for p.Y > 999 {
		p.Y -= 1000
		canvasIdx += 2
	}

	client := &http.Client{}
	data := fmt.Sprintf(
		`{"operationName": "setPixel", "variables": {"input": {"actionName": "r/replace:set_pixel", "PixelMessageData": {"coordinate": {"x": %d, "y": %d}, "colorIndex": %d, "canvasIndex": %d}}}, "query": "mutation setPixel($input: ActInput!) {\n  act(input: $input) {\n    data {\n      ... on BasicMessage {\n        id\n        data {\n          ... on GetUserCooldownResponseMessageData {\n            nextAvailablePixelTimestamp\n            __typename\n          }\n          ... on SetPixelResponseMessageData {\n            timestamp\n            __typename\n          }\n          __typename\n        }\n        __typename\n      }\n      __typename\n    }\n    __typename\n  }\n}\n"}`,
		p.X, p.Y, colorIdx, canvasIdx,
	)
	body := bytes.NewBuffer([]byte(data))
	req, err := http.NewRequest("POST", "https://gql-realtime-2.reddit.com/query", body)
	if err != nil {
		return fmt.Errorf("error creating new request: %v", err)
	}

	req.Header.Add("accept", "*/*")
	req.Header.Add("accept-language", "en,en-US;q=0.9,he;q=0.8")
	req.Header.Add("apollographql-client-name", "mona-lisa")
	req.Header.Add("apollographql-client-version", "0.0.1")
	req.Header.Add("authorization", fmt.Sprintf("Bearer %s", a.token))
	req.Header.Add("content-type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error executing request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("%s token: %s", a.Username, a.token)
		return fmt.Errorf("error code %d, %s", resp.StatusCode, resp.Status)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	respBodyS := string(respBody)

	if respBodyS[2:8] == "errors" {
		_, msgTrailer, _ := strings.Cut(respBodyS, `"message":"`)
		msg, _, _ := strings.Cut(msgTrailer, `"`)

		_, msgTrailer, _ = strings.Cut(msgTrailer, `nextAvailablePixelTs":`)
		nextAvailablePixelTs, _, _ := strings.Cut(msgTrailer, `}`)
		nextAvailableMs, err := strconv.Atoi(nextAvailablePixelTs)
		if err != nil {
			return err
		}
		t := time.UnixMilli(int64(nextAvailableMs))
		a.nextAvailableTime = t
		return fmt.Errorf("error placing pixel: %s, next available placement at: %s", msg, t.Format(time.RFC3339))
	}

	_, msgTrailer, _ := strings.Cut(respBodyS, `nextAvailablePixelTimestamp":`)
	nextAvailablePixelTimestamp, _, _ := strings.Cut(msgTrailer, `,`)
	nextAvailablePixelFl, err := strconv.ParseFloat(nextAvailablePixelTimestamp, 64)
	if err != nil {
		return err
	}
	nextAvailableMs := int(nextAvailablePixelFl)
	t := time.UnixMilli(int64(nextAvailableMs))
	a.nextAvailableTime = t
	return nil
}

func decodeImg(filePath string) (image.Image, error) {
	f, err := os.OpenFile(filePath, os.O_RDONLY, 0400)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	return png.Decode(f)
}

// getMisplacedPixels returns a slice of Points that need correction.
func getMisplacedPixels(img image.Image, at image.Point) ([]rplace.Update, error) {
	ctx := morecontext.ForSignals()
	defer ctx.Done()
	c := rplace.Client{}
	return c.GetDiff(ctx, img, at)
}

// getUpdateChan returns a channel of pixel updates.
func getUpdateChan(img image.Image, at image.Point) (chan rplace.Update, error) {
	ctx := morecontext.ForSignals()
	cli := rplace.Client{}
	return cli.NeededUpdatesFor(ctx, img, at)
}

func main() {

	flag.Parse()
	filepath := flag.Arg(0)
	img, err := decodeImg(filepath)
	if err != nil {
		log.Fatalf("failed to decode image: %v", err)
	}

	jsonFilename := "config.json"
	jsonFile, err := os.Open(jsonFilename)
	defer jsonFile.Close()
	if err != nil {
		log.Fatalf("failed to open json file %s, error: %v", jsonFilename, err)
	}
	jsonData, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Fatalf("failed to read json file, error: %v", err)
	}

	cfg := Config{}
	if err := json.Unmarshal(jsonData, &cfg); err != nil {
		log.Fatalf("failed to unmarshal json file, error: %v", err)
		return
	}

	c := cfg.Cli
	a := cfg.Accounts

	at := image.Point{X: cfg.X, Y: cfg.Y}

	updateCh, err := getUpdateChan(img, at)
	if err != nil {
		log.Fatalf("failed to get update channel: %v", err)
	}
	for up := range updateCh {
		// walk all accounts to find one not on cooldown
		for i := 0; i < len(a); i++ {
			acct := &a[i]
			// skip account if it's still on cooldown
			if time.Now().UnixNano() <= acct.nextAvailableTime.UnixNano() {
				log.Printf("%s still on cooldown until %s", acct.Username, acct.nextAvailableTime.Format(time.RFC3339))
				continue
			}
			// check if token needs refresh
			if time.Now().UnixNano() > acct.tokenExpiry.UnixNano() {
				// refresh token
				err = acct.getAuthToken(c)
				if err != nil {
					log.Fatalf("failed to get auth token: %v", err)
					continue
				}
			}

			err = acct.setPixel(up.Point, colorMap[up.Color.Name])
			if err != nil {
				log.Printf("%s : %v", acct.Username, err)
				continue
			}
			log.Printf("%s : Wrote %s to {%d,%d}", acct.Username, up.Color.Name, up.Point.X, up.Point.Y)
			time.Sleep(time.Second)
			break
		}
		// Builtin random cooldown, anywhere from 0 to 1499ms.
		extraCooldown := rand.Int63n(1500)
		time.Sleep(time.Duration(cfg.SleepSeconds)*time.Second + time.Duration(extraCooldown)*time.Millisecond)
	}
}
