package sinmetalcraft

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/net/context"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

type AppConfig struct {
	ClientId                string    `json:"clientId" datastore:",noindex"`                // GCP Client Id
	ClientSecret            string    `json:"clientSecret" datastore:",noindex"`            // GCP Client Secret
	SlackPostUrl            string    `json:"slackPostUrl" datastore:",noindex"`            // Slackにぶっこむ用URL
	APIAIIntentIDRunServer  string    `json:"aPIAIIntentIDRunServer" datastore:",noindex"`  // api.ai RunServerのIntentID
	APIAIIntentIDItemRecipe string    `json:"aPIAIIntentIDItemRecipe" datastore:",noindex"` // api.ai レシピ検索のIntentID
	CreatedAt               time.Time `json:"createdAt"`                                    // 作成日時
	UpdatedAt               time.Time `json:"updatedAt"`                                    // 更新日時
}

const (
	appConfigId = "app-config-id"
)

type AppConfigApi struct {
}

type AppConfigService struct {
}

func init() {
	api := AppConfigApi{}

	http.HandleFunc("/admin/api/1/config", api.Handler)
}

func (a *AppConfigApi) Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		a.Put(w, r)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (a *AppConfigApi) Put(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	var ac AppConfig
	err := json.NewDecoder(r.Body).Decode(&ac)
	if err != nil {
		log.Infof(ctx, "rquest body, %v", r.Body)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	_, err = datastore.Put(ctx, datastore.NewKey(ctx, "AppConfig", appConfigId, 0, nil), &ac)
	if err != nil {
		log.Errorf(ctx, fmt.Sprintf("datastore put error : ", err.Error()))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ac)
}

func (s *AppConfigService) Get(ctx context.Context) (AppConfig, error) {
	key := datastore.NewKey(ctx, "AppConfig", appConfigId, 0, nil)
	var ac AppConfig
	err := datastore.Get(ctx, key, &ac)
	return ac, err
}

func (ac *AppConfig) Load(ps []datastore.Property) error {
	if err := datastore.LoadStruct(ac, ps); err != nil {
		return err
	}

	return nil
}

func (ac *AppConfig) Save() ([]datastore.Property, error) {
	now := time.Now()
	ac.UpdatedAt = now

	if ac.CreatedAt.IsZero() {
		ac.CreatedAt = now
	}

	return datastore.SaveStruct(ac)
}
