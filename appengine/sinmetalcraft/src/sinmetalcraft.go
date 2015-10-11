package sinmetalcraft

import (
	"fmt"
	"io/ioutil"
	"net/http"

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
	http.HandleFunc("/api/1/minecraft", api.Post)
}

type MinecraftApi struct{}

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
	log.Infof(ctx, string(body))

	w.WriteHeader(200)
}

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
