package sinmetalcraft

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"google.golang.org/appengine/user"

	"google.golang.org/api/compute/v1"

	"golang.org/x/net/context"
)

const PROJECT_NAME = "sinmetalcraft"
const INSTANCE_NAME = "minecraft"

func init() {
	api := MinecraftApi{}

	http.HandleFunc("/minecraft", handlerMinecraftLog)
	http.HandleFunc("/api/1/minecraft", api.Handler)
}

type Minecraft struct {
	Key             *datastore.Key `json:"-" datastore:"-"`
	KeyStr          string         `json:"key" datastore:"-"`
	World           string         `json:"world"`
	ResourceID      int64          `json:"resourceID"`
	Zone            string         `json:"zone" datastore:",unindexed"`
	IPAddr          string         `json:"ipAddr" datastore:",unindexed"`
	Status          string         `json:"status" datastore:",unindexed"`
	OperationType   string         `json:"operationType" datastore:",unindexed"`
	OperationStatus string         `json:"operationstatus" datastore:",unindexed"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

type MinecraftApiListResponse struct {
	Items  []MinecraftApiResponse `json:"items"`
	Cursor string                 `json:cursor`
}

type MinecraftApiResponse struct {
	InstanceName      string `json:"instanceName"`
	Zone              string `json:"zone"`
	IPAddr            string `json:"iPAddr"`
	Status            string `json:"status"`
	CreationTimestamp string `json:"creationTimestamp"`
}

type Metadata struct {
	ProjectID   string            `json:"projectId"`
	ServiceName string            `json:"serviceName"`
	Zone        string            `json:"zone"`
	Labels      map[string]string `json:"labels"`
	Timestamp   string            `json:"timestamp"`
}

type StructPayload struct {
	Log string `json:"log"`
}

type PubSubData struct {
	Metadata      Metadata      `json:"metadata"`
	InsertID      string        `json:"insertId"`
	Log           string        `json:"log"`
	StructPayload StructPayload `json:"structPayload"`
}

type Message struct {
	Data       string            `json:"data"`
	Attributes map[string]string `json:"attributes"`
	MessageID  string            `json:"message_id"`
}

type PubSubBody struct {
	Message      Message `json:"message"`
	Subscription string  `json:"subscription"`
}

type MinecraftApi struct{}

// /api/1/minecraft handler
func (a *MinecraftApi) Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		a.Post(w, r)
	} else if r.Method == "PUT" {
		a.Put(w, r)
	} else if r.Method == "GET" {
		a.List(w, r)
	} else if r.Method == "DELETE" {
		a.Delete(w, r)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// create world data
func (a *MinecraftApi) Post(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	u := user.Current(ctx)
	if u == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		loginURL, err := user.LoginURL(ctx, "")
		if err != nil {
			log.Errorf(ctx, "get user login URL error, %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(fmt.Sprintf(`{"loginURL":"%s"}`, loginURL)))
		return
	}
	if user.IsAdmin(ctx) == false {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var minecraft Minecraft
	err := json.NewDecoder(r.Body).Decode(&minecraft)
	if err != nil {
		log.Infof(ctx, "rquest body, %v", r.Body)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "invalid request."}`))
		return
	}
	defer r.Body.Close()

	key := datastore.NewKey(ctx, "Minecraft", minecraft.World, 0, nil)
	err = datastore.RunInTransaction(ctx, func(c context.Context) error {
		var entity Minecraft
		err := datastore.Get(ctx, key, &entity)
		if err != datastore.ErrNoSuchEntity && err != nil {
			return err
		}

		minecraft.Status = "no_exists"
		now := time.Now()
		minecraft.CreatedAt = now
		minecraft.UpdatedAt = now
		_, err = datastore.Put(ctx, key, &minecraft)
		if err != nil {
			return err
		}

		return nil
	}, nil)
	if err != nil {
		log.Errorf(ctx, "Minecraft Put Error. error = %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	minecraft.KeyStr = key.Encode()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(minecraft)
}

// update world data
func (a *MinecraftApi) Put(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	u := user.Current(ctx)
	if u == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		loginURL, err := user.LoginURL(ctx, "")
		if err != nil {
			log.Errorf(ctx, "get user login URL error, %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(fmt.Sprintf(`{"loginURL":"%s"}`, loginURL)))
		return
	}
	if user.IsAdmin(ctx) == false {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	var minecraft Minecraft
	err := json.NewDecoder(r.Body).Decode(&minecraft)
	if err != nil {
		log.Infof(ctx, "rquest body, %v", r.Body)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "invalid request."}`))
		return
	}
	defer r.Body.Close()

	key, err := datastore.DecodeKey(minecraft.KeyStr)
	if err != nil {
		log.Infof(ctx, "invalid key, %v", r.Body)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "invalid key."}`))
		return
	}
	minecraft.Key = key

	err = datastore.RunInTransaction(ctx, func(c context.Context) error {
		var entity Minecraft
		err := datastore.Get(ctx, key, &entity)
		if err != datastore.ErrNoSuchEntity && err != nil {
			return err
		}

		entity.IPAddr = minecraft.IPAddr
		entity.Zone = minecraft.Zone
		entity.UpdatedAt = time.Now()
		_, err = datastore.Put(ctx, key, &minecraft)
		if err != nil {
			return err
		}

		return nil
	}, nil)
	if err != nil {
		log.Errorf(ctx, "Minecraft Put Error. error = %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(minecraft)
}

// delete world data
func (a *MinecraftApi) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	u := user.Current(ctx)
	if u == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		loginURL, err := user.LoginURL(ctx, "")
		if err != nil {
			log.Errorf(ctx, "get user login URL error, %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write([]byte(fmt.Sprintf(`{"loginURL":"%s"}`, loginURL)))
		return
	}
	if user.IsAdmin(ctx) == false {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	keyStr := r.FormValue("key")

	key, err := datastore.DecodeKey(keyStr)
	if err != nil {
		log.Infof(ctx, "invalid key, %v", r.Body)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "invalid key."}`))
		return
	}

	err = datastore.RunInTransaction(ctx, func(c context.Context) error {
		return datastore.Delete(ctx, key)
	}, nil)
	if err != nil {
		log.Errorf(ctx, "Minecraft Delete Error. error = %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(`{}`)
}

// list world data
func (a *MinecraftApi) List(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	q := datastore.NewQuery("Minecraft").Order("-UpdatedAt")

	list := make([]*Minecraft, 0)
	for t := q.Run(ctx); ; {
		var entity Minecraft
		key, err := t.Next(&entity)
		if err == datastore.Done {
			break
		}
		if err != nil {
			log.Errorf(ctx, "Minecraft Query Error. error = %s", err.Error())
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		entity.Key = key
		entity.KeyStr = key.Encode()
		list = append(list, &entity)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(list)
}

// handle cloud pub/sub request
func handlerMinecraftLog(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	for k, v := range r.Header {
		log.Infof(ctx, "%s:%s", k, v)
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf(ctx, "ERROR request body read: %s", err)
		w.WriteHeader(500)
		return
	}
	log.Infof(ctx, "request body = %s", string(body))

	var psb PubSubBody
	err = psb.Decode(body)
	if err != nil {
		log.Errorf(ctx, "ERROR request body Pub Sub Body decode: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Infof(ctx, "request Pub Sub Body = %v", psb)

	var psd PubSubData
	err = psd.Decode(psb.Message.Data)
	if err != nil {
		log.Errorf(ctx, "ERROR request body Pub Sub Data decode: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Infof(ctx, "request Pub Sub Data = %v", psd)

	var sm SlackMessage
	fields := make([]SlackField, 0)

	sa := SlackAttachment{
		Color:      "#36a64f",
		AuthorName: "sinmetalcraft",
		AuthorIcon: "https://storage.googleapis.com/sinmetalcraft-image/minecraft.jpeg",
		Title:      psd.StructPayload.Log,
		Fields:     fields,
	}

	sm.UserName = "sinmetalcraft"
	sm.IconUrl = "https://storage.googleapis.com/sinmetalcraft-image/minecraft.jpeg"
	sm.Text = psd.StructPayload.Log
	sm.Attachments = []SlackAttachment{sa}

	acs := AppConfigService{}
	config, err := acs.Get(ctx)
	if err != nil {
		log.Errorf(ctx, "ERROR App Config Get: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = PostToSlack(ctx, config.SlackPostUrl, sm)
	if err != nil {
		log.Errorf(ctx, "ERROR Post Slack: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// list gce instance
func listInstance(ctx context.Context, is *compute.InstancesService, zone string) ([]*compute.Instance, string, error) {
	ilc := is.List(PROJECT_NAME, zone)
	il, err := ilc.Do()
	if err != nil {
		return nil, "", err
	}
	return il.Items, il.NextPageToken, nil
}

// create gce instance
func createInstance(ctx context.Context, is *compute.InstancesService, minecraft Minecraft) (string, error) {
	name := INSTANCE_NAME + "-" + minecraft.World
	log.Infof(ctx, "create instance name = %s", name)

	startupScriptURL := "gs://sinmetalcraft-minecraft-shell/minecraftserver-startup-script.sh"
	shutdownScriptURL := "gs://sinmetalcraft-minecraft-shell/minecraftserver-shutdown-script.sh"
	stateValue := "new"
	newIns := &compute.Instance{
		Name:        name,
		Zone:        "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/zones/" + minecraft.Zone,
		MachineType: "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/zones/" + minecraft.Zone + "/machineTypes/n1-standard-4",
		Disks: []*compute.AttachedDisk{
			&compute.AttachedDisk{
				AutoDelete: true,
				Boot:       true,
				DeviceName: name,
				Mode:       "READ_WRITE",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/global/images/minecraft-image-v20151114a",
					DiskType:    "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/zones/" + minecraft.Zone + "/diskTypes/pd-ssd",
					DiskSizeGb:  100,
				},
			},
		},
		CanIpForward: false,
		NetworkInterfaces: []*compute.NetworkInterface{
			&compute.NetworkInterface{
				Network: "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/global/networks/default",
				AccessConfigs: []*compute.AccessConfig{
					&compute.AccessConfig{
						Name:  "External NAT",
						Type:  "ONE_TO_ONE_NAT",
						NatIP: minecraft.IPAddr,
					},
				},
			},
		},
		Tags: &compute.Tags{
			Items: []string{
				"minecraft-server",
			},
		},
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{
				&compute.MetadataItems{
					Key:   "startup-script-url",
					Value: &startupScriptURL,
				},
				&compute.MetadataItems{
					Key:   "shutdown-script-url",
					Value: &shutdownScriptURL,
				},
				&compute.MetadataItems{
					Key:   "world",
					Value: &minecraft.World,
				},
				&compute.MetadataItems{
					Key:   "state",
					Value: &stateValue,
				},
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			&compute.ServiceAccount{
				Email: "default",
				Scopes: []string{
					compute.DevstorageReadWriteScope,
					"https://www.googleapis.com/auth/logging.write",
				},
			},
		},
		Scheduling: &compute.Scheduling{
			AutomaticRestart:  false,
			OnHostMaintenance: "TERMINATE",
			Preemptible:       true,
		},
	}
	ope, err := is.Insert(PROJECT_NAME, minecraft.Zone, newIns).Do()
	if err != nil {
		log.Errorf(ctx, "ERROR insert instance: %s", err)
		return "", err
	}
	WriteLog(ctx, "INSTNCE_CREATE_OPE", ope)

	_, err = CallMinecraftTQ(ctx, minecraft.Key, ope.Name)
	if err != nil {
		return name, err
	}

	return name, nil
}

// start instance
func startInstance(ctx context.Context, is *compute.InstancesService, minecraft Minecraft) (string, error) {
	name := INSTANCE_NAME + "-" + minecraft.World
	log.Infof(ctx, "start instance name = %s", name)

	ope, err := is.Start(PROJECT_NAME, minecraft.Zone, name).Do()
	if err != nil {
		log.Errorf(ctx, "ERROR reset instance: %s", err)
		return "", err
	}
	WriteLog(ctx, "INSTNCE_START_OPE", ope)

	_, err = CallMinecraftTQ(ctx, minecraft.Key, ope.Name)
	if err != nil {
		return name, err
	}

	return name, nil
}

// reset instance
func resetInstance(ctx context.Context, is *compute.InstancesService, minecraft Minecraft) (string, error) {
	name := INSTANCE_NAME + "-" + minecraft.World
	log.Infof(ctx, "reset instance name = %s", name)

	ope, err := is.Reset(PROJECT_NAME, minecraft.Zone, name).Do()
	if err != nil {
		log.Errorf(ctx, "ERROR reset instance: %s", err)
		return "", err
	}
	WriteLog(ctx, "INSTNCE_RESET_OPE", ope)

	_, err = CallMinecraftTQ(ctx, minecraft.Key, ope.Name)
	if err != nil {
		return name, err
	}

	return name, nil
}

// delete instance
func deleteInstance(ctx context.Context, is *compute.InstancesService, minecraft Minecraft) (string, error) {
	name := INSTANCE_NAME + "-" + minecraft.World
	log.Infof(ctx, "delete instance name = %s", name)

	ope, err := is.Delete(PROJECT_NAME, minecraft.Zone, name).Do()
	if err != nil {
		log.Errorf(ctx, "ERROR delete instance: %s", err)
		return "", err
	}
	WriteLog(ctx, "INSTNCE_DELETE_OPE", ope)

	_, err = CallMinecraftTQ(ctx, minecraft.Key, ope.Name)
	if err != nil {
		return name, err
	}

	return name, nil
}

func (psb *PubSubBody) Decode(body []byte) error {
	err := json.Unmarshal(body, psb)
	if err != nil {
		return err
	}
	return nil

}

func (psd *PubSubData) Decode(body string) error {
	mr := base64.NewDecoder(base64.StdEncoding, strings.NewReader(body))
	return json.NewDecoder(mr).Decode(psd)
}

type SlackMessage struct {
	UserName    string            `json:"username"`
	IconUrl     string            `json:"icon_url"`
	Text        string            `json:"text"`
	Attachments []SlackAttachment `json:"attachments"`
}

type SlackAttachment struct {
	Color      string       `json:"color"`
	AuthorName string       `json:"author_name"`
	AuthorLink string       `json:"author_link"`
	AuthorIcon string       `json:"author_icon"`
	Title      string       `json:"title"`
	TitleLink  string       `json:"title_link"`
	Fields     []SlackField `json:"fields"`
}

type SlackField struct {
	Title string `json:"title"`
}

func PostToSlack(ctx context.Context, url string, message SlackMessage) (resp *http.Response, err error) {
	client := urlfetch.Client(ctx)

	body, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(body))
	return client.Post(
		url,
		"application/json",
		bytes.NewReader(body))
}

func WriteLog(ctx context.Context, key string, v interface{}) {
	body, err := json.Marshal(v)
	if err != nil {
		log.Errorf(ctx, "WriteLog Error %s %v", err.Error(), v)
	}
	log.Infof(ctx, `{"%s":%s}`, key, body)
}
