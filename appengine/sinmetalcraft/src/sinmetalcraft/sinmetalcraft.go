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
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"

	"google.golang.org/api/compute/v1"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const PROJECT_NAME = "sinmetalcraft"
const INSTANCE_NAME = "minecraft"

func init() {
	api := MinecraftApi{}

	http.HandleFunc("/minecraft", handlerMinecraftLog)
	http.HandleFunc("/api/1/minecraft", api.Handler)
}

type Minecraft struct {
	World           string    `json:"world"`
	ResourceID      int64     `json:"resourceID"`
	IPAddr          string    `json:"ipAddr" datastore:",unindexed"`
	Status          string    `json:"status" datastore:",unindexed"`
	OperationType   string    `json:"operationType" datastore:",unindexed"`
	OperationStatus string    `json:"operationstatus" datastore:",unindexed"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
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
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// create instance
func (a *MinecraftApi) Post(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	client := &http.Client{
		Transport: &oauth2.Transport{
			Source: google.AppEngineTokenSource(ctx, compute.ComputeScope),
			Base:   &urlfetch.Transport{Context: ctx},
		},
	}
	s, err := compute.New(client)
	if err != nil {
		log.Errorf(ctx, "ERROR compute.New: %s", err)
		w.WriteHeader(500)
		return
	}
	is := compute.NewInstancesService(s)
	name, err := createInstance(ctx, is, "minecraft", "asia-east1-b", "104.155.205.121")
	if err != nil {
		log.Errorf(ctx, "ERROR compute.New: %s", err)
		w.WriteHeader(500)
		return
	}

	w.WriteHeader(200)
	w.Write([]byte(fmt.Sprintf("%s create done!", name)))
}

// reset or start instance
func (a *MinecraftApi) Put(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	ope := r.FormValue("operation")
	if ope != "start" && ope != "reset" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"invalid request" : "operation param is start or reset"}`))
		return
	}

	client := &http.Client{
		Transport: &oauth2.Transport{
			Source: google.AppEngineTokenSource(ctx, compute.ComputeScope),
			Base:   &urlfetch.Transport{Context: ctx},
		},
	}
	s, err := compute.New(client)
	if err != nil {
		log.Errorf(ctx, "ERROR compute.New: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	is := compute.NewInstancesService(s)

	var name string
	if ope == "start" {
		name, err = startInstance(ctx, is, "minecraft", "asia-east1-b")
		if err != nil {
			log.Errorf(ctx, "ERROR compute Instances Start: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else if ope == "reset" {
		name, err = resetInstance(ctx, is, "minecraft", "asia-east1-b")
		if err != nil {
			log.Errorf(ctx, "ERROR compute Instances Reset: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("%s %s done!", name, ope)))
}

func (a *MinecraftApi) List(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	client := &http.Client{
		Transport: &oauth2.Transport{
			Source: google.AppEngineTokenSource(ctx, compute.ComputeScope),
			Base:   &urlfetch.Transport{Context: ctx},
		},
	}
	s, err := compute.New(client)
	if err != nil {
		log.Errorf(ctx, "ERROR compute.New: %s", err)
		w.WriteHeader(500)
		return
	}
	is := compute.NewInstancesService(s)
	instances, cursor, err := listInstance(ctx, is, "asia-east1-b")
	if err != nil {
		log.Errorf(ctx, "ERROR compute.Instance List: %s", err)
		w.WriteHeader(500)
		return
	}

	var res []MinecraftApiResponse
	for _, item := range instances {
		res = append(res, MinecraftApiResponse{
			InstanceName:      item.Name,
			Zone:              item.Zone,
			IPAddr:            item.NetworkInterfaces[0].AccessConfigs[0].NatIP,
			Status:            item.Status,
			CreationTimestamp: item.CreationTimestamp,
		})
	}

	apiRes := MinecraftApiListResponse{
		Items:  res,
		Cursor: cursor,
	}
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(apiRes)
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
func createInstance(ctx context.Context, is *compute.InstancesService, world string, zone string, ipAddr string) (string, error) {
	name := INSTANCE_NAME + "-" + world
	log.Infof(ctx, "create instance name = %s", name)

	startupScriptURL := "gs://sinmetalcraft-minecraft-shell/minecraftserver-startup-script.sh"
	shutdownScriptURL := "gs://sinmetalcraft-minecraft-shell/minecraftserver-shutdown-script.sh"
	newIns := &compute.Instance{
		Name:        name,
		Zone:        "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/zones/" + zone,
		MachineType: "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/zones/" + zone + "/machineTypes/n1-highcpu-2",
		Disks: []*compute.AttachedDisk{
			&compute.AttachedDisk{
				AutoDelete: true,
				Boot:       true,
				DeviceName: name,
				Mode:       "READ_WRITE",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/global/images/minecraft-image-v20151012",
					DiskType:    "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/zones/" + zone + "/diskTypes/pd-ssd",
					DiskSizeGb:  50,
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
						NatIP: ipAddr,
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
					Value: &world,
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
	ope, err := is.Insert(PROJECT_NAME, zone, newIns).Do()
	if err != nil {
		log.Errorf(ctx, "ERROR insert instance: %s", err)
		return "", err
	}
	WriteLog(ctx, "INSTNCE_CREATE_OPE", ope)

	_, err = CallMinecraftTQ(ctx, world, ipAddr, ope.Name)
	if err != nil {
		return name, err
	}

	return name, nil
}

// start instance
func startInstance(ctx context.Context, is *compute.InstancesService, world string, zone string) (string, error) {
	name := INSTANCE_NAME + "-" + world
	log.Infof(ctx, "start instance name = %s", name)

	ope, err := is.Start(PROJECT_NAME, zone, name).Do()
	if err != nil {
		log.Errorf(ctx, "ERROR reset instance: %s", err)
		return "", err
	}
	WriteLog(ctx, "INSTNCE_START_OPE", ope)

	// TODO ipAddr
	_, err = CallMinecraftTQ(ctx, world, "", ope.Name)
	if err != nil {
		return name, err
	}

	return name, nil
}

// reset instance
func resetInstance(ctx context.Context, is *compute.InstancesService, world string, zone string) (string, error) {
	name := INSTANCE_NAME + "-" + world
	log.Infof(ctx, "reset instance name = %s", name)

	ope, err := is.Reset(PROJECT_NAME, zone, name).Do()
	if err != nil {
		log.Errorf(ctx, "ERROR reset instance: %s", err)
		return "", err
	}
	WriteLog(ctx, "INSTNCE_RESET_OPE", ope)

	// TODO ipAddr
	_, err = CallMinecraftTQ(ctx, world, "", ope.Name)
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
