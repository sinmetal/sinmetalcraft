package sinmetalcraft

import (
	"net/http"
	"strings"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"

	"google.golang.org/api/compute/v1"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func init() {
	api := MinecraftCronApi{}

	http.HandleFunc("/cron/1/minecraft/vacuum", api.Handler)
}

type MinecraftCronApi struct{}

// /cron/1/minecraft/vacuum handler
func (a *MinecraftCronApi) Handler(w http.ResponseWriter, r *http.Request) {
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

	instances, _, err := listInstance(ctx, is, "asia-east1-b")
	if err != nil {
		log.Errorf(ctx, "ERROR list instance error %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	api := MinecraftCronApi{}
	receiver := make([]<-chan error, 0)
	for _, ins := range instances {
		log.Infof(ctx, "Instance Name = %s, Status = %s", ins.Name, ins.Status)

		if strings.HasPrefix(ins.Name, "minecraft-") {
			log.Infof(ctx, `%s has prefix "minecraft-"`, ins.Name)
			if ins.Status == "TERMINATED" {
				ch := api.deleteInstance(ctx, is, ins.Name[len("minecraft-"):len(ins.Name)])
				receiver = append(receiver, ch)
			}
		} else {
			log.Infof(ctx, `%s has not prefix "minecraft-"`, ins.Name)
		}
	}
	var hasError bool
	for _, ch := range receiver {
		err := <-ch
		if err != nil {
			hasError = true
			log.Errorf(ctx, "ERROR : Delete Instance Error = %s", err.Error())
		}
	}

	if hasError {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func (a *MinecraftCronApi) deleteInstance(ctx context.Context, is *compute.InstancesService, world string) <-chan error {
	log.Infof(ctx, "Delete Instance Target World Name = %s", world)

	receiver := make(chan error)
	go func() {
		var minecraft Minecraft
		key := datastore.NewKey(ctx, "Minecraft", world, 0, nil)

		err := datastore.Get(ctx, key, &minecraft)
		if err == datastore.ErrNoSuchEntity {
			log.Infof(ctx, "Minecraft Entity Not Found. world = %s", world)
			receiver <- nil
		}
		if err != nil {
			receiver <- nil
		}
		minecraft.Key = key

		_, err = deleteInstance(ctx, is, minecraft)
		receiver <- err
	}()
	return receiver
}
