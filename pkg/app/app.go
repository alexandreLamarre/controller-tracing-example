package app

import (
	"context"

	"github.com/rancher/lasso/pkg/cache"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	"github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	corecontroller "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/v3/pkg/generic"
	"github.com/rancher/wrangler/v3/pkg/start"
	"k8s.io/client-go/rest"
)

func controllerFactory(rest *rest.Config) (controller.SharedControllerFactory, error) {
	clientFactory, err := client.NewSharedClientFactory(rest, nil)
	if err != nil {
		return nil, err
	}

	cacheFactory := cache.NewSharedCachedFactory(clientFactory, nil)
	return controller.NewSharedControllerFactory(cacheFactory, &controller.SharedControllerFactoryOptions{
		DefaultWorkers: 10,
	}), nil
}

type AppContext struct {
	Core corecontroller.Interface

	scf      controller.SharedControllerFactory
	starters []start.Starter
}

func Setup(_ context.Context, client *rest.Config) (*AppContext, error) {
	scf, err := controllerFactory(client)
	if err != nil {
		return nil, err
	}

	core, err := core.NewFactoryFromConfigWithOptions(client, &generic.FactoryOptions{
		SharedControllerFactory: scf,
	})
	if err != nil {
		return nil, err
	}
	corev := core.Core().V1()

	return &AppContext{
		Core: corev,
		scf:  scf,
		starters: []start.Starter{
			core,
		},
	}, nil
}

func (a *AppContext) Start(ctx context.Context) error {

	if err := a.scf.Start(ctx, 10); err != nil {
		return err
	}
	return start.All(ctx, 10, a.starters...)
}
