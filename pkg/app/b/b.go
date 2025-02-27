package b

import (
	"context"
	"log/slog"
	"strings"

	"github.com/alexandreLamarre/controller-tracing-example/pkg/app"
	"github.com/rancher/lasso/pkg/tracing"
	v1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"go.opentelemetry.io/otel"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	tracer = otel.GetTracerProvider().Tracer("operatorB")
)

type handler struct {
	logger *slog.Logger

	namespaces v1.NamespaceController
	configMaps v1.ConfigMapController
}

func (h *handler) setupCtx(ctx context.Context, obj runtime.Object) context.Context {
	return tracing.Extract(ctx, obj)
}

func (h *handler) OnNamespaceRemove(key string, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	h.logger.With("key", key).Debug("remove queued")
	return namespace, nil
}

func (h *handler) OnNamespaceChange(key string, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	ctx := h.setupCtx(context.TODO(), namespace)
	spanCtx, span := tracer.Start(ctx, "OnNamespaceChange")
	defer span.End()
	logger := h.logger.With("key", key)
	logger.Debug("on change")
	if namespace == nil {
		logger.Debug("namespace is nil")
		return nil, nil
	}
	op, ok := namespace.Labels["controller-tracing-example"]
	if !ok {
		logger.Info("no operation found")
		return namespace, nil
	}
	logger.With("op", op).Debug("operation found")
	if strings.ToLower(op) == "b" {
		logger.Info("creating config map")

		cfg := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "controller-tracing-example",
				Namespace: namespace.Name,
			},
			Data: map[string]string{
				"controller-tracing-example": "B",
			},
		}
		tracing.Inject(spanCtx, cfg)
		_, err := h.configMaps.Create(cfg)
		if err != nil {
			logger.Error(err.Error())
		}
		return namespace, nil
	}
	h.configMaps.Enqueue(namespace.Namespace, "controller-tracing-example")
	return namespace, nil
}

func Register(ctx context.Context, app *app.AppContext) {
	logger := slog.Default().With("controller", "B")
	namespaces := app.Core.Namespace()
	configMaps := app.Core.ConfigMap()

	h := &handler{
		logger:     logger,
		namespaces: namespaces,
		configMaps: configMaps,
	}
	// namespaces.OnRemove(ctx, "namespace-remove-B", h.OnNamespaceRemove)
	namespaces.OnChange(ctx, "namespace-change-b", h.OnNamespaceChange)
}
