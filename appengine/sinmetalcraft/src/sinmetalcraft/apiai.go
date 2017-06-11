package sinmetalcraft

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

// ApiAIApi is api.aiからのRequestを処理するAPI
type ApiAIApi struct{}

func init() {
	api := ApiAIApi{}

	http.HandleFunc("/apiai", api.handler)
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
	w.WriteHeader(http.StatusOK)

	sm := &struct {
		Text string `json:"text"`
	}{
		Text: "ここがSlackか",
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
