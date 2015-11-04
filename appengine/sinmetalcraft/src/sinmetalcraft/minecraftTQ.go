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

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func init() {
	api := MinecraftTQApi{}

	http.HandleFunc("/tq/1/minecraft", api.Handler)
}

type MinecraftTQApi struct{}

func CallMinecraftTQ(c context.Context, minecraftKey *datastore.Key, operationID string) (*taskqueue.Task, error) {
	t := taskqueue.NewPOSTTask("/tq/1/minecraft", url.Values{
		"keyStr":      {minecraftKey.Encode()},
		"operationID": {operationID},
	})
	t.Delay = time.Second * 30
	return taskqueue.Add(c, t, "minecraft")
}

// /tq/1/minecraft handler
func (a *MinecraftTQApi) Handler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	keyStr := r.FormValue("KeyStr")
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

	status := "exists"
	if ope.OperationType == "delete" {
		status = "not_exists"
	}

	resStatus := http.StatusOK
	if ope.Status == "DONE" {
		err = datastore.RunInTransaction(ctx, func(c context.Context) error {
			var entity Minecraft
			err := datastore.Get(ctx, key, &entity)
			if err != nil {
				return err
			}

			entity.ResourceID = int64(ope.TargetId)
			entity.Status = status
			entity.OperationStatus = ope.Status
			entity.OperationType = ope.OperationType
			entity.UpdatedAt = time.Now()

			_, err = datastore.Put(ctx, key, &entity)
			if err != nil {
				return err
			}

			return nil
		}, nil)
	} else {
		log.Infof(ctx, "Operation Status = %s", ope.Status)
		resStatus = http.StatusRequestTimeout

		err = datastore.RunInTransaction(ctx, func(c context.Context) error {
			var entity Minecraft
			err := datastore.Get(ctx, key, &entity)
			if err != nil {
				return err
			}

			entity.ResourceID = 0
			entity.OperationStatus = ope.Status
			entity.OperationType = ope.OperationType
			entity.UpdatedAt = time.Now()

			_, err = datastore.Put(ctx, key, &entity)
			if err != nil {
				return err
			}

			return nil
		}, nil)
	}
	if err != nil {
		log.Errorf(ctx, "Minecraft Put Error. error = %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(resStatus)
}
