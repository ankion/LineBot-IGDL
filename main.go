package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/line/line-bot-sdk-go/linebot"
)

const (
	IgTypeNone = 0 + iota
	IgTypeImage
	IgTypeVideo
)

func main() {
	bot, err := linebot.New(
		os.Getenv("CHANNEL_SECRET"),
		os.Getenv("CHANNEL_TOKEN"),
	)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		events, err := bot.ParseRequest(req)
		if err != nil {
			if err == linebot.ErrInvalidSignature {
				w.WriteHeader(400)
			} else {
				w.WriteHeader(500)
			}
			return
		}
		for _, event := range events {
			if event.Type == linebot.EventTypeMessage {
				switch message := event.Message.(type) {
				case *linebot.TextMessage:
					if strings.Contains(message.Text, "instagram.com") {
						log.Println("[event] msg:" + message.Text)
						igType, result, err := parseIG(message.Text)
						if err != nil {
							log.Println(err)
							return
						}
						switch igType {
						case IgTypeImage:
							var replys []linebot.Message
							for i, imgURL := range result {
								if i >= 5 {
									break
								}
								replys = append(replys, linebot.NewImageMessage(imgURL, imgURL))
							}
							if _, err = bot.ReplyMessage(event.ReplyToken, replys...).Do(); err != nil {
								log.Println(err)
								return
							}
						case IgTypeVideo:
							if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewVideoMessage(result[1], result[0])).Do(); err != nil {
								log.Println(err)
								return
							}
						}

					}
				}
			}
		}
	})
	// This is just sample code.
	// For actual use, you must support HTTPS by using `ListenAndServeTLS`, a reverse proxy or something else.
	log.Println("Starting service on port:" + os.Getenv("PORT"))
	if err := http.ListenAndServe(":"+os.Getenv("PORT"), nil); err != nil {
		log.Fatal(err)
	}
}

func parseIG(igURL string) (int, []string, error) {
	if strings.Contains(igURL, "instgram.com") {
		return IgTypeNone, nil, errors.New("not instgram url")
	}
	resp, err := http.Get(igURL)
	if err != nil {
		return IgTypeNone, nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	re := regexp.MustCompile(`<meta property=\"og:image\" content=\"(.*)\" />`)
	baseURL := re.FindStringSubmatch(string(body))
	if len(baseURL) < 1 {
		return IgTypeNone, nil, errors.New("No such base url")
	}

	re = regexp.MustCompile(`<meta property="og:video:secure_url" content="(.*)" />`)
	baseVideoURL := re.FindStringSubmatch(string(body))
	if len(baseVideoURL) > 0 {
		result := []string{baseURL[1], baseVideoURL[1]}
		return IgTypeVideo, result, nil
	}
	baseDomain := regexp.MustCompile(`/\d+_\d+_\d+_n\.jpg`).Split(baseURL[1], -1)
	if len(baseDomain) < 1 {
		return IgTypeNone, nil, errors.New("No such base domain")
	}

	re = regexp.MustCompile(`"display_url": "(` + baseDomain[0] + `/\d+_\d+_\d+_n\.jpg)",`)
	photoURLs := re.FindAllStringSubmatch(string(body), -1)
	result := []string{}
	for _, v := range photoURLs {
		if yes, _ := Contain(v[1], result); !yes {
			result = append(result, v[1])
		}
	}

	return IgTypeImage, result, nil
}

func Contain(obj interface{}, target interface{}) (bool, error) {
	targetValue := reflect.ValueOf(target)
	switch reflect.TypeOf(target).Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < targetValue.Len(); i++ {
			if targetValue.Index(i).Interface() == obj {
				return true, nil
			}
		}
	case reflect.Map:
		if targetValue.MapIndex(reflect.ValueOf(obj)).IsValid() {
			return true, nil
		}
	}

	return false, errors.New("not in array")
}
