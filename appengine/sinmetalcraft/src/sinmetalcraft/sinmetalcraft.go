package sinmetalcraft

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

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
	log.Infof(ctx, "instance name = %s", name)

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
					SourceImage: "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/global/images/minecraft-image",
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
					Value: "gs://sinmetalcraft-minecraft-shell/minecraftserver-startup-script.sh",
				},
				&compute.MetadataItems{
					Key:   "shutdown-script-url",
					Value: "gs://sinmetalcraft-minecraft-shell/minecraftserver-shutdown-script.sh",
				},
				&compute.MetadataItems{
					Key:   "world",
					Value: world,
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
	log.Infof(ctx, "create instance ope.name = %s, ope.targetLink = %s, ope.Status = %s", ope.Name, ope.TargetLink, ope.Status)

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
