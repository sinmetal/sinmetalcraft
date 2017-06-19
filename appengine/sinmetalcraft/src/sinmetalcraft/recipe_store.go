package sinmetalcraft

import (
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
)

type Recipe struct {
	Key *datastore.Key `json:"key" datastore:"-"`
	Recipe []string `datestore:",unindexed"`
}

func (r *Recipe) RecipeString() string {
	return ""
}

type RecipeStore struct {
}

func (s *RecipeStore) Get(c context.Context, id string) (*Recipe, error) {
	key := datastore.NewKey(c, "Recipe", id, 0, nil)

	var recipe Recipe
	err := datastore.Get(c, key, &recipe)
	if err != nil {
		return nil, err
	}
	return &recipe, nil
}