package sinmetalcraft

import (
	"time"

	"google.golang.org/appengine/datastore"

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
