package sinmetalcraft

import (
	"net/http"
	"net/url"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/taskqueue"
	"google.golang.org/appengine/urlfetch"

	"google.golang.org/api/compute/v1"

	"errors"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func init() {
	api := ServerTQApi{}

	http.HandleFunc("/tq/1/server/instance/create", api.CreateInstance)
	http.HandleFunc("/tq/1/server/instance/delete", api.DeleteInstance)
}

type ServerTQApi struct{}

func (a *ServerTQApi) CallCreateInstance(c context.Context, minecraftKey *datastore.Key, operationID string) (*taskqueue.Task, error) {
	log.Infof(c, "Call Minecraft TQ, key = %v, operationID = %s", minecraftKey, operationID)
	if minecraftKey == nil {
		return nil, errors.New("key is required")
	}
	if len(operationID) < 1 {
		return nil, errors.New("operationID is required")
	}

	t := taskqueue.NewPOSTTask("/tq/1/server/instance/create", url.Values{
		"keyStr":      {minecraftKey.Encode()},
		"operationID": {operationID},
	})
	t.Delay = time.Second * 30
	return taskqueue.Add(c, t, "minecraft")
}

func (a *ServerTQApi) CallDeleteInstance(c context.Context, minecraftKey *datastore.Key, operationID string, latestSnapshot string) (*taskqueue.Task, error) {
	log.Infof(c, "Call Minecraft TQ, key = %v, operationID = %s", minecraftKey, operationID)
	if minecraftKey == nil {
		return nil, errors.New("key is required")
	}
	if len(operationID) < 1 {
		return nil, errors.New("operationID is required")
	}

	t := taskqueue.NewPOSTTask("/tq/1/server/instance/delete", url.Values{
		"keyStr":         {minecraftKey.Encode()},
		"operationID":    {operationID},
		"latestSnapshot": {latestSnapshot},
	})
	t.Delay = time.Second * 30
	return taskqueue.Add(c, t, "minecraft")
}

func (a *ServerTQApi) CreateInstance(w http.ResponseWriter, r *http.Request) {
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
	name, err := createInstance(ctx, is, entity)
	if err != nil {
		log.Errorf(ctx, "instance create error. error = %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Infof(ctx, "instance create done. name = %s", name)
	w.WriteHeader(http.StatusOK)
}

func (a *ServerTQApi) DeleteInstance(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	keyStr := r.FormValue("keyStr")
	operationID := r.FormValue("operationID")
	latestSnapshot := r.FormValue("latestSnapshot")

	log.Infof(ctx, "keyStr = %s, operationID = %s, latestSnapshot = %s", keyStr, operationID, latestSnapshot)

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
	err = datastore.RunInTransaction(ctx, func(c context.Context) error {
		err := datastore.Get(ctx, key, &entity)
		if err != nil {
			return err
		}

		entity.LatestSnapshot = latestSnapshot
		entity.UpdatedAt = time.Now()

		_, err = datastore.Put(ctx, key, &entity)
		if err != nil {
			return err
		}

		return nil
	}, nil)
	entity.Key = key

	is := compute.NewInstancesService(s)
	name, err := deleteInstance(ctx, is, entity)
	if err != nil {
		log.Errorf(ctx, "instance delete error. error = %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Infof(ctx, "instance delete done. name = %s", name)
	w.WriteHeader(http.StatusOK)
}
