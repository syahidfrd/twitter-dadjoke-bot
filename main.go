package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/dghubble/oauth1"
	"github.com/gorilla/mux"
	_ "github.com/joho/godotenv/autoload"
)

type TwitterMentionHook struct {
	ForUserID         string `json:"for_user_id"`
	UserHasBlocked    bool   `json:"user_has_blocked"`
	TweetCreateEvents []struct {
		CreatedAt            string      `json:"created_at"`
		ID                   int64       `json:"id"`
		IDStr                string      `json:"id_str"`
		Text                 string      `json:"text"`
		Source               string      `json:"source"`
		Truncated            bool        `json:"truncated"`
		InReplyToStatusID    interface{} `json:"in_reply_to_status_id"`
		InReplyToStatusIDStr interface{} `json:"in_reply_to_status_id_str"`
		InReplyToUserID      interface{} `json:"in_reply_to_user_id"`
		InReplyToUserIDStr   interface{} `json:"in_reply_to_user_id_str"`
		InReplyToScreenName  interface{} `json:"in_reply_to_screen_name"`
		User                 struct {
			ID                             int64         `json:"id"`
			IDStr                          string        `json:"id_str"`
			Name                           string        `json:"name"`
			ScreenName                     string        `json:"screen_name"`
			Location                       interface{}   `json:"location"`
			URL                            interface{}   `json:"url"`
			Description                    interface{}   `json:"description"`
			TranslatorType                 string        `json:"translator_type"`
			Protected                      bool          `json:"protected"`
			Verified                       bool          `json:"verified"`
			FollowersCount                 int           `json:"followers_count"`
			FriendsCount                   int           `json:"friends_count"`
			ListedCount                    int           `json:"listed_count"`
			FavouritesCount                int           `json:"favourites_count"`
			StatusesCount                  int           `json:"statuses_count"`
			CreatedAt                      string        `json:"created_at"`
			UtcOffset                      interface{}   `json:"utc_offset"`
			TimeZone                       interface{}   `json:"time_zone"`
			GeoEnabled                     bool          `json:"geo_enabled"`
			Lang                           interface{}   `json:"lang"`
			ContributorsEnabled            bool          `json:"contributors_enabled"`
			IsTranslator                   bool          `json:"is_translator"`
			ProfileBackgroundColor         string        `json:"profile_background_color"`
			ProfileBackgroundImageURL      string        `json:"profile_background_image_url"`
			ProfileBackgroundImageURLHTTPS string        `json:"profile_background_image_url_https"`
			ProfileBackgroundTile          bool          `json:"profile_background_tile"`
			ProfileLinkColor               string        `json:"profile_link_color"`
			ProfileSidebarBorderColor      string        `json:"profile_sidebar_border_color"`
			ProfileSidebarFillColor        string        `json:"profile_sidebar_fill_color"`
			ProfileTextColor               string        `json:"profile_text_color"`
			ProfileUseBackgroundImage      bool          `json:"profile_use_background_image"`
			ProfileImageURL                string        `json:"profile_image_url"`
			ProfileImageURLHTTPS           string        `json:"profile_image_url_https"`
			DefaultProfile                 bool          `json:"default_profile"`
			DefaultProfileImage            bool          `json:"default_profile_image"`
			Following                      interface{}   `json:"following"`
			FollowRequestSent              interface{}   `json:"follow_request_sent"`
			Notifications                  interface{}   `json:"notifications"`
			WithheldInCountries            []interface{} `json:"withheld_in_countries"`
		} `json:"user"`
		Geo           interface{} `json:"geo"`
		Coordinates   interface{} `json:"coordinates"`
		Place         interface{} `json:"place"`
		Contributors  interface{} `json:"contributors"`
		IsQuoteStatus bool        `json:"is_quote_status"`
		QuoteCount    int         `json:"quote_count"`
		ReplyCount    int         `json:"reply_count"`
		RetweetCount  int         `json:"retweet_count"`
		FavoriteCount int         `json:"favorite_count"`
		Entities      struct {
			Hashtags     []interface{} `json:"hashtags"`
			Urls         []interface{} `json:"urls"`
			UserMentions []struct {
				ScreenName string `json:"screen_name"`
				Name       string `json:"name"`
				ID         int64  `json:"id"`
				IDStr      string `json:"id_str"`
				Indices    []int  `json:"indices"`
			} `json:"user_mentions"`
			Symbols []interface{} `json:"symbols"`
		} `json:"entities"`
		Favorited   bool   `json:"favorited"`
		Retweeted   bool   `json:"retweeted"`
		FilterLevel string `json:"filter_level"`
		Lang        string `json:"lang"`
		TimestampMs string `json:"timestamp_ms"`
	} `json:"tweet_create_events"`
}

type DadJoke struct {
	ID     string `json:"id"`
	Joke   string `json:"joke"`
	Status int    `json:"status"`
}

func getRandomDadJoke() (dadJoke DadJoke, err error) {
	url := "https://icanhazdadjoke.com/"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}

	req.Header.Add("Accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	if res.StatusCode >= 400 {
		err = fmt.Errorf("the request api dadjoke returned status code %d", res.StatusCode)
		return
	}

	if err = json.Unmarshal(body, &dadJoke); err != nil {
		return
	}

	return
}

func crcHandler(rw http.ResponseWriter, r *http.Request) {
	crcToken := r.URL.Query()["crc_token"]
	if len(crcToken) < 1 {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("no crc token given"))
		return
	}

	h := hmac.New(sha256.New, []byte(os.Getenv("TWITTER_CONSUMER_SECRET")))
	h.Write([]byte(crcToken[0]))
	sha := base64.StdEncoding.EncodeToString(h.Sum(nil))
	responseToken := fmt.Sprintf("sha256=%s", sha)

	js, _ := json.Marshal(map[string]interface{}{
		"response_token": responseToken,
	})

	rw.Header().Set("Content-Type", "application/json")
	rw.Write(js)
}

func wehbookHandler(rw http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(rw, "cannot read response body", http.StatusBadRequest)
		return
	}

	var rawData map[string]interface{}
	if err := json.Unmarshal(body, &rawData); err != nil {
		http.Error(rw, "invalid parsing body response - rawData", http.StatusBadRequest)
		return
	}

	if _, exist := rawData["tweet_create_events"]; !exist {
		rw.WriteHeader(http.StatusOK)
		return
	}

	var twitterMentionHook TwitterMentionHook
	if err := json.Unmarshal(body, &twitterMentionHook); err != nil {
		http.Error(rw, "invalid parsing body response - twitterMentionHook", http.StatusBadRequest)
		return
	}

	text := twitterMentionHook.TweetCreateEvents[0].Text
	replayID := twitterMentionHook.TweetCreateEvents[0].IDStr
	username := twitterMentionHook.TweetCreateEvents[0].User.ScreenName

	dadJokeMatch, _ := regexp.MatchString("\\#dadjoke\\b", text)
	if dadJokeMatch {
		log.Println("received #dadjoke request")
		log.Println(string(body))

		res, err := getRandomDadJoke()
		if err != nil {
			fmt.Println(err.Error())
			http.Error(rw, "failed get dadjoke", http.StatusBadRequest)
			return
		}

		status := fmt.Sprintf("@%s %s", username, res.Joke)
		if err := replyTweet(status, replayID); err != nil {
			fmt.Println(err.Error())
			http.Error(rw, "failed to replay tweet", http.StatusBadRequest)
			return
		}

	}

	// TODO - Other random stuff

	rw.Write([]byte("ok"))
}

func twitterClient() (client *http.Client) {
	config := oauth1.NewConfig(os.Getenv("TWITTER_CONSUMER_KEY"), os.Getenv("TWITTER_CONSUMER_SECRET"))
	token := oauth1.NewToken(os.Getenv("TWITTER_ACCESS_TOKEN"), os.Getenv("TWITTER_ACCESS_TOKEN_SECRET"))
	client = config.Client(oauth1.NoContext, token)
	return
}

func replyTweet(tweet string, replyID string) (err error) {
	path := fmt.Sprintf("%s/statuses/update.json", os.Getenv("TWITTER_BASE_URL"))

	params := url.Values{}
	params.Set("status", tweet)
	params.Set("in_reply_to_status_id", replyID)

	client := twitterClient()
	resp, err := client.PostForm(path, params)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var data map[string]interface{}
	if err = json.Unmarshal([]byte(body), &data); err != nil {
		return
	}

	if resp.StatusCode >= 400 {
		err = fmt.Errorf("the request api reply tweet returned status code: %d", resp.StatusCode)
		return
	}

	return
}

func main() {

	r := mux.NewRouter()

	r.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("server up and running!"))
	})

	r.HandleFunc("/webhook/twitter", crcHandler).Methods("GET")
	r.HandleFunc("/webhook/twitter", wehbookHandler).Methods("POST")

	srv := &http.Server{
		Handler:      r,
		Addr:         ":" + os.Getenv("PORT"),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
