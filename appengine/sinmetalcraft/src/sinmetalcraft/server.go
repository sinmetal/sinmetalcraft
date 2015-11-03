package sinmetalcraft

import (
	"encoding/json"
	"fmt"
	"net/http"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"

	"google.golang.org/api/compute/v1"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func init() {
	api := ServerApi{}

	http.HandleFunc("/api/1/server", api.Handler)
}

type ServerApi struct{}

func (a *ServerApi) Handler(w http.ResponseWriter, r *http.Request) {
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

// create new instance
func (a *ServerApi) Post(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	var form map[string]string
	err := json.NewDecoder(r.Body).Decode(&form)
	if err != nil {
		log.Infof(ctx, "rquest body, %v", r.Body)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "invalid request."}`))
		return
	}
	defer r.Body.Close()

	if len(form["key"]) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "key is required."}`))
		return
	}
	key, err := datastore.DecodeKey(form["key"])
	if err != nil {
		log.Infof(ctx, "decode key error. param = %s, err = %s", form["key"], err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "invalid key."}`))
		return
	}

	var minecraft Minecraft
	err = datastore.Get(ctx, key, &minecraft)
	if err == datastore.ErrNoSuchEntity {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf(`{"message": "%s is not found."}`, form["key"])))
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"message": "%s"}`, err.Error())))
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
	name, err := createInstance(ctx, is, minecraft.World, minecraft.Zone, minecraft.IPAddr)
	if err != nil {
		log.Errorf(ctx, "ERROR compute instances create: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf(`{"message" : "%s create done!"}`, name)))
}

// reset or start instance
func (a *ServerApi) Put(w http.ResponseWriter, r *http.Request) {
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

func (a *ServerApi) Delete(w http.ResponseWriter, r *http.Request) {
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
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	is := compute.NewInstancesService(s)

	name, err := deleteInstance(ctx, is, "minecraft", "asia-east1-b")
	if err != nil {
		log.Errorf(ctx, "ERROR compute Instances Start: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("%s delete done!", name)))
}

func (a *ServerApi) List(w http.ResponseWriter, r *http.Request) {
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
