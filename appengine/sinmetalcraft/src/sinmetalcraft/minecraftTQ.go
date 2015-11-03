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

func CallMinecraftTQ(c context.Context, world string, ipAddr string, operationID string) (*taskqueue.Task, error) {
	t := taskqueue.NewPOSTTask("/tq/1/minecraft", url.Values{
		"world":       {world},
		"ipAddr":      {ipAddr},
		"operationID": {operationID},
	})
	t.Delay = time.Second * 30
	return taskqueue.Add(c, t, "minecraft")
}

// /tq/1/minecraft handler
func (a *MinecraftTQApi) Handler(w http.ResponseWriter, r *http.Request) {
	ctx := appengine.NewContext(r)

	world := r.FormValue("world")
	ipAddr := r.FormValue("ipAddr")
	operationID := r.FormValue("operationID")

	log.Infof(ctx, "world = %s, idAddr = %s, operationID = %s", world, ipAddr, operationID)

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
	}
	WriteLog(ctx, "__GET_ZONE_COMPUTE_OPE__", ope)

	resStatus := http.StatusOK
	key := datastore.NewKey(ctx, "Minecraft", world, 0, nil)
	if ope.Status == "DONE" {
		err = datastore.RunInTransaction(ctx, func(c context.Context) error {
			var entity Minecraft
			err := datastore.Get(ctx, key, &entity)
			if err == datastore.ErrNoSuchEntity {
				entity.World = world
				entity.CreatedAt = time.Now()
			} else if err != nil {
				return err
			}

			entity.ResourceID = int64(ope.TargetId)
			entity.IPAddr = ipAddr
			entity.Status = "" // TODO
			entity.OperationStatus = ope.Status
			entity.OperationType = ope.OperationType
			entity.UpdatedAt = time.Now()

			_, err = datastore.Put(ctx, key, &entity)
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
	} else {
		log.Infof(ctx, "Operation Status != DONE")
		resStatus = http.StatusRequestTimeout

		err = datastore.RunInTransaction(ctx, func(c context.Context) error {
			var entity Minecraft
			err := datastore.Get(ctx, key, &entity)
			if err == datastore.ErrNoSuchEntity {
				entity.World = world
				entity.CreatedAt = time.Now()
			} else if err != nil {
				return err
			}

			entity.ResourceID = 0
			entity.IPAddr = ""
			entity.Status = "" // TODO
			entity.OperationStatus = ope.Status
			entity.OperationType = ope.OperationType
			entity.UpdatedAt = time.Now()

			_, err = datastore.Put(ctx, key, &entity)
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
	}

	w.WriteHeader(resStatus)
}
