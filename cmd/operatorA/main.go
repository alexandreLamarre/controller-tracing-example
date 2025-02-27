package main

import (
	"log"
	"os"

	"github.com/alexandreLamarre/controller-tracing-example/pkg/app"
	"github.com/alexandreLamarre/controller-tracing-example/pkg/app/a"
	_ "github.com/alexandreLamarre/controller-tracing-example/pkg/logger"
	"github.com/alexandreLamarre/controller-tracing-example/pkg/tracing"
	"github.com/rancher/wrangler/v3/pkg/kubeconfig"
	"github.com/rancher/wrangler/v3/pkg/ratelimit"
	"github.com/rancher/wrangler/v3/pkg/signals"
)

func main() {
	if err := tracing.Init("operatorA"); err != nil {
		log.Fatalf("failed to init tracing")
	}
	ctx := signals.SetupSignalContext()
	k := os.Getenv("KUBECONFIG")
	if k == "" {
		log.Fatal("no kubeconfig set")
	}
	restKubeConfig := kubeconfig.GetNonInteractiveClientConfig(k)
	clientCmd, err := restKubeConfig.ClientConfig()
	if err != nil {
		log.Fatalf("failed to get client config: %v", err)
	}
	clientCmd.RateLimiter = ratelimit.None

	appCtx, err := app.Setup(ctx, clientCmd)
	if err != nil {
		log.Fatalf("failed to setup app: %v", err)
	}
	a.Register(ctx, appCtx)
	if err := appCtx.Start(ctx); err != nil {
		log.Fatalf("failed to start app: %v", err)
	}

	<-ctx.Done()
}
