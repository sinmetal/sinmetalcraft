package sinmetalcraft

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

// ApiAIApi is api.aiからのRequestを処理するAPI
type ApiAIApi struct{}

func init() {
	api := ApiAIApi{}

	http.HandleFunc("/apiai", api.handler)
}

// APIARequest is api.aiから飛んでくるRequest
type APIAIRequest struct {
	OriginalRequest OriginalRequest `json:"originalRequest"`
	ID              string          `json:"id"`
	Timestamp       string          `json:"timestamp"`
	Lang            string          `json:"lang"`
	Result          Result          `json:"result"`
	Status          APIAIStatus     `json:"status"`
	SessionID       string          `json:"sessionId"`
}

type Result struct {
	Source           string            `json:"source"`
	ResolvedQuery    string            `json:"resolvedQuery"`
	Speech           string            `json:"speech"`
	Action           string            `json:"action"`
	ActionInComplete bool              `json:"actionIncomplete"`
	Parameters       map[string]string `json:"parameters"`
	Contexts         []string          `json:"contexts"`
	Metadata         APIAIMetadata     `json:"metadata"`
	Fulfillment      Fulfillment       `json:"fulfillment"`
	Score            float64           `json:"score"`
}

type APIAIStatus struct {
	Code      int    `json:"code"`
	ErrorType string `json:"errorType"`
}

type APIAIMetadata struct {
	IntentID                  string `json:"intentId"`
	WebhookUsed               string `json:"webhookUsed"`
	WebhookForSlotFillingUsed string `json:"webhookForSlotFillingUsed"`
	IntentName                string `json:"intentName"`
}

type Fulfillment struct {
	Speech   string         `json:"speech"`
	Messages []APIAIMessage `json:"messages"`
}

type APIAIMessage struct {
	Type   int    `json:"type"`
	Speech string `json:"speech"`
}

type OriginalRequest struct {
	Source string `json:"source"`
	Data   Data   `json:"data"`
}

type Data struct {
	AuthedUsers []string `json:"authed_users"`
	EventID     string   `json:"event_id"`
	APIAppID    string   `json:"api_app_id"`
	TeamID      string   `json:"team_id"`
	Event       Event    `json:"event"`
	Type        string   `json:"type"`
	EventTime   int64    `json:"event_time"`
	Token       string   `json:"token"`
}

type Event struct {
	EventTimestamp string `json:"event_ts"`
	Channel        string `json:"channel"`
	Text           string `json:"text"`
	Type           string `json:"type"`
	User           string `json:"user"`
	Timestamp      string `json:"ts"`
}

// APIAIResponse is api.aiからのRequestに返すResponse
type APIAIResponse struct {
	Speech        string        `json:"speech"`
	DisplayText   string        `json:"displayText"`
	Data          interface{}   `json:"data"`
	ContextOut    []interface{} `json:"contextOut"`
	Source        string        `json:"source"`
	FollowupEvent interface{}   `json:"followupEvent"`
}

func (a *ApiAIApi) handler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	for k, v := range r.Header {
		log.Infof(c, "%s = %s", k, v)
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf(c, "%s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Infof(c, "%s", b)

	var req APIAIRequest
	err = json.Unmarshal(b, &req)
	if err != nil {
		log.Errorf(c, "%s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var responseText = "ここがSlackか"
	var configService AppConfigService
	config, err := configService.Get(c)
	if err != nil {
		log.Errorf(c, "%s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if req.Result.Metadata.IntentID == config.APIAIIntentIDRunServer {
		var m Minecraft
		l, err := m.QueryExistsServers(c)
		if err != nil {
			log.Errorf(c, "%s", err.Error())
		} else {
			if len(l) > 0 {
				var worlds = make([]string, len(l), len(l))
				for i, v := range l {
					worlds[i] = v.World
				}
				text := strings.Join(worlds[:], ",")
				responseText = fmt.Sprintf("起動しているのは `%s` だよ！", text)
			} else {
				responseText = "起動しているサーバはないみたい"
			}

		}
	}

	w.WriteHeader(http.StatusOK)

	sm := &struct {
		Text string `json:"text"`
	}{
		Text: responseText,
	}
	slack := &struct {
		Slack interface{} `json:"slack"`
	}{
		Slack: sm,
	}

	res := APIAIResponse{
		Data:   slack,
		Source: "DuckDuckGo",
	}
	err = json.NewEncoder(w).Encode(res)
	if err != nil {
		log.Errorf(c, "%s", err.Error())
	}
}
