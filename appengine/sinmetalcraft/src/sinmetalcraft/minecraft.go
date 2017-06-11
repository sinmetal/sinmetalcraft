package sinmetalcraft

import (
	"time"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"

	"golang.org/x/net/context"
)

// UpdateOverviewerSnapshot is Overviewerを作成したSnapshotのVersionを更新する
func (m *Minecraft) UpdateOverviewerSnapshot(ctx context.Context, key *datastore.Key) error {
	return datastore.RunInTransaction(ctx, func(c context.Context) error {
		var entity Minecraft
		err := datastore.Get(ctx, key, &entity)
		if err != datastore.ErrNoSuchEntity && err != nil {
			return err
		}

		entity.OverviewerSnapshot = entity.LatestSnapshot
		entity.UpdatedAt = time.Now()
		_, err = datastore.Put(ctx, key, &entity)
		if err != nil {
			return err
		}

		return nil
	}, nil)
}

//QueryExistsServers is 起動しているサーバ一覧を取得
func (m *Minecraft) QueryExistsServers(ctx context.Context) ([]Minecraft, error) {
	var minecrafts []Minecraft
	_, err := datastore.NewQuery("Minecraft").Filter("Status = ", "exists").GetAll(ctx, &minecrafts)
	if err != nil {
		log.Errorf(ctx, "Minecraft Query error. %s\n", err.Error())
	}
	return minecrafts, nil
}
