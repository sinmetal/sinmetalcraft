package sinmetalcraft

import (
	"encoding/json"
	"fmt"
	"net/http"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"
	"google.golang.org/appengine/user"

	"google.golang.org/api/compute/v1"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func init() {
	api := ServerApi{}

	http.HandleFunc("/api/1/server", api.Handler)
}

type ServerApi struct{}

type ServerApiPutParam struct {
	KeyStr    string `json:"key"`
	Operation string `json:"operation"`
}

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
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "invalid request."}`))
		return
	}
	defer r.Body.Close()

	if len(form["key"]) < 1 {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "key is required."}`))
		return
	}
	key, err := datastore.DecodeKey(form["key"])
	if err != nil {
		log.Infof(ctx, "decode key error. param = %s, err = %s", form["key"], err.Error())
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "invalid key."}`))
		return
	}

	var minecraft Minecraft
	err = datastore.Get(ctx, key, &minecraft)
	if err == datastore.ErrNoSuchEntity {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf(`{"message": "%s is not found."}`, form["key"])))
		return
	}
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"message": "%s"}`, err.Error())))
		return
	}
	minecraft.Key = key

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
	ds := compute.NewDisksService(s)
	ope, err := createDiskFromSnapshot(ctx, ds, minecraft)
	if err != nil {
		log.Errorf(ctx, "ERROR create disk: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	stqAPI := ServerTQApi{}
	_, err = stqAPI.CallCreateInstance(ctx, minecraft.Key, ope.Name)
	if err != nil {
		log.Errorf(ctx, "ERROR call create instance tq: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf(`{"message" : "%s create done!"}`, minecraft.World)))
}

// reset or start instance
func (a *ServerApi) Put(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	var param ServerApiPutParam
	err := json.NewDecoder(r.Body).Decode(&param)
	if err != nil {
		log.Infof(ctx, "rquest body, %v", r.Body)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "invalid request."}`))
		return
	}
	defer r.Body.Close()

	key, err := datastore.DecodeKey(param.KeyStr)
	if err != nil {
		log.Infof(ctx, "invalid key, %v", r.Body)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "invalid key."}`))
		return
	}

	if param.Operation != "start" && param.Operation != "reset" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"invalid request" : "operation param is start or reset"}`))
		return
	}

	var minecraft Minecraft
	err = datastore.Get(ctx, key, &minecraft)
	if err == datastore.ErrNoSuchEntity {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf(`{"message": "%s is not found."}`, param.KeyStr)))
		return
	}
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf(`{"message": "%s"}`, err.Error())))
		return
	}
	minecraft.Key = key

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
	if param.Operation == "start" {
		name, err = startInstance(ctx, is, minecraft)
		if err != nil {
			log.Errorf(ctx, "ERROR compute Instances Start: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else if param.Operation == "reset" {
		name, err = resetInstance(ctx, is, minecraft)
		if err != nil {
			log.Errorf(ctx, "ERROR compute Instances Reset: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		log.Errorf(ctx, `{"invalid request" : "operation param is start or reset"}`)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"invalid request" : "operation param is start or reset"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"message": "%s %s done!"}`, name, param.Operation)))
}

// delete instance
func (a *ServerApi) Delete(w http.ResponseWriter, r *http.Request) {
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

	keyStr := r.FormValue("key")

	key, err := datastore.DecodeKey(keyStr)
	if err != nil {
		log.Infof(ctx, "invalid key, %v", r.Body)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message": "invalid key."}`))
		return
	}

	var minecraft Minecraft
	err = datastore.Get(ctx, key, &minecraft)
	if err == datastore.ErrNoSuchEntity {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf(`{"message": "%s is not found."}`, keyStr)))
		return
	}
	if err != nil {
		log.Errorf(ctx, "ERROR, Get Minecraft error = %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	minecraft.Key = key

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

	name, err := deleteInstance(ctx, is, minecraft)
	if err != nil {
		log.Errorf(ctx, "ERROR compute Instances Delete: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`{"message": "%s delete done!"}`, name)))
}

// list instance
func (a *ServerApi) List(w http.ResponseWriter, r *http.Request) {
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
	instances, cursor, err := listInstance(ctx, is, "asia-northeast1-b")
	if err != nil {
		log.Errorf(ctx, "ERROR compute.Instance List: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
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

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiRes)
}
