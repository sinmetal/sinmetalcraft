package sinmetalcraft

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/taskqueue"
	"google.golang.org/appengine/urlfetch"

	"google.golang.org/api/compute/v1"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type OverviewerAPI struct{}

const OverviewerInstanceName = "minecraft-overviewer"
const OverViewerWorldDiskFormat = "%s-overviewer-world-%s"

func init() {
	api := OverviewerAPI{}

	http.HandleFunc("/cron/1/overviewer", api.handler)
	http.HandleFunc("/tq/1/overviewer/instance/create", api.handleTQCreateInstace)
}

func (a *OverviewerAPI) handler(w http.ResponseWriter, r *http.Request) {
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

	client := &http.Client{
		Transport: &oauth2.Transport{
			Source: google.AppEngineTokenSource(ctx, compute.ComputeScope),
			Base:   &urlfetch.Transport{Context: ctx},
		},
	}

	// TODO 本来は1つずつTQにする方がよい
	for _, minecraft := range list {
		if minecraft.LatestSnapshot == minecraft.OverviewerSnapshot {
			continue
		}

		s, err := compute.New(client)
		if err != nil {
			log.Errorf(ctx, "ERROR compute.New: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		ds := compute.NewDisksService(s)
		ope, err := a.createDiskFromSnapshot(ctx, ds, *minecraft)
		if err != nil {
			log.Errorf(ctx, "ERROR create disk: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = a.CallCreateInstance(ctx, minecraft.Key, ope.Name)
		if err != nil {
			log.Errorf(ctx, "ERROR call create instance tq: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = minecraft.UpdateOverviewerSnapshot(ctx, minecraft.Key)
		if err != nil {
			log.Errorf(ctx, "ERROR Update OverviewerSnapshot: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
}

// create disk from snapshot
func (a *OverviewerAPI) createDiskFromSnapshot(ctx context.Context, ds *compute.DisksService, minecraft Minecraft) (*compute.Operation, error) {
	name := fmt.Sprintf(OverViewerWorldDiskFormat, INSTANCE_NAME, minecraft.World)
	d := &compute.Disk{
		Name:           name,
		SizeGb:         100,
		SourceSnapshot: "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/global/snapshots/" + minecraft.LatestSnapshot,
		Type:           "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/zones/" + minecraft.Zone + "/diskTypes/pd-ssd",
	}

	ope, err := ds.Insert(PROJECT_NAME, minecraft.Zone, d).Do()
	if err != nil {
		log.Errorf(ctx, "ERROR insert disk: %s", err)
		return nil, err
	}
	WriteLog(ctx, "INSTNCE_DISK_OPE", ope)

	return ope, err
}

// create gce instance
func (a *OverviewerAPI) createInstance(ctx context.Context, is *compute.InstancesService, minecraft Minecraft) (string, error) {
	name := OverviewerInstanceName + "-" + minecraft.World
	worldDiskName := fmt.Sprintf(OverViewerWorldDiskFormat, INSTANCE_NAME, minecraft.World)
	log.Infof(ctx, "create instance name = %s", name)

	startupScriptURL := "gs://sinmetalcraft-minecraft-shell/minecraft-overviewer-server-startup-script.sh"
	shutdownScriptURL := "gs://sinmetalcraft-minecraft-shell/minecraft-overviewer-shutdown-script.sh"
	stateValue := "new"
	newIns := &compute.Instance{
		Name:        name,
		Zone:        "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/zones/" + minecraft.Zone,
		MachineType: "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/zones/" + minecraft.Zone + "/machineTypes/n1-highcpu-4",
		Disks: []*compute.AttachedDisk{
			&compute.AttachedDisk{
				AutoDelete: true,
				Boot:       true,
				DeviceName: name,
				Mode:       "READ_WRITE",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/global/images/family/minecraft-overviewer",
					DiskType:    "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/zones/" + minecraft.Zone + "/diskTypes/pd-ssd",
					DiskSizeGb:  100,
				},
			},
			&compute.AttachedDisk{
				AutoDelete: true,
				Boot:       false,
				DeviceName: worldDiskName,
				Mode:       "READ_WRITE",
				Source:     "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/zones/" + minecraft.Zone + "/disks/" + worldDiskName,
			},
		},
		CanIpForward: false,
		NetworkInterfaces: []*compute.NetworkInterface{
			&compute.NetworkInterface{
				Network: "https://www.googleapis.com/compute/v1/projects/" + PROJECT_NAME + "/global/networks/default",
				AccessConfigs: []*compute.AccessConfig{
					&compute.AccessConfig{
						Name: "External NAT",
						Type: "ONE_TO_ONE_NAT",
					},
				},
			},
		},
		Tags: &compute.Tags{
			Items: []string{},
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
				&compute.MetadataItems{
					Key:   "minecraft-version",
					Value: &minecraft.JarVersion,
				},
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			&compute.ServiceAccount{
				Email: "default",
				Scopes: []string{
					compute.DevstorageReadWriteScope,
					compute.ComputeScope,
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

	return name, nil
}

func (a *OverviewerAPI) CallCreateInstance(c context.Context, minecraftKey *datastore.Key, operationID string) (*taskqueue.Task, error) {
	log.Infof(c, "Call Minecraft TQ, key = %v, operationID = %s", minecraftKey, operationID)
	if minecraftKey == nil {
		return nil, errors.New("key is required")
	}
	if len(operationID) < 1 {
		return nil, errors.New("operationID is required")
	}

	t := taskqueue.NewPOSTTask("/tq/1/overviewer/instance/create", url.Values{
		"keyStr":      {minecraftKey.Encode()},
		"operationID": {operationID},
	})
	t.Delay = time.Second * 30
	return taskqueue.Add(c, t, "minecraft")
}

// handleTQCreateInstace is Overviewerのためにインスタンスを作成するためのTQ Handler
// Snapshotから復元されたDiskが作成されたのを確認してから、インスタンスを作成する
func (a *OverviewerAPI) handleTQCreateInstace(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	keyStr := r.FormValue("keyStr")
	operationID := r.FormValue("operationID")

	log.Infof(ctx, "keyStr = %s, operationID = %s", keyStr, operationID)

	key, err := datastore.DecodeKey(keyStr)
	if err != nil {
		log.Errorf(ctx, "key decode error. keyStr = %s, err = %s", keyStr, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
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
	nzos := compute.NewZoneOperationsService(s)
	ope, err := nzos.Get(PROJECT_NAME, "asia-east1-b", operationID).Do()
	if err != nil {
		log.Errorf(ctx, "ERROR compute Zone Operation Get Error. zone = %s, operation = %s, error = %s", "asia-east1-b", operationID, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	WriteLog(ctx, "__GET_ZONE_COMPUTE_OPE__", ope)

	if ope.Status != "DONE" {
		log.Infof(ctx, "operation status = %s", ope.Status)
		w.WriteHeader(http.StatusRequestTimeout)
		return
	}

	var entity Minecraft
	err = datastore.Get(ctx, key, &entity)
	if err != nil {
		log.Errorf(ctx, "datastore get error. key = %s. error = %v", key.StringID(), err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	entity.Key = key

	is := compute.NewInstancesService(s)
	name, err := a.createInstance(ctx, is, entity)
	if err != nil {
		log.Errorf(ctx, "instance create error. error = %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Infof(ctx, "instance create done. name = %s", name)
	w.WriteHeader(http.StatusOK)
}
