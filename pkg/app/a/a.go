package a

import (
	"context"
	"log/slog"
	"strings"

	"github.com/alexandreLamarre/controller-tracing-example/pkg/app"
	"github.com/rancher/lasso/pkg/tracing"
	v1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"go.opentelemetry.io/otel"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	tracer = otel.GetTracerProvider().Tracer("operatorA")
)

type handler struct {
	logger *slog.Logger

	namespaces v1.NamespaceController
}

func (h *handler) setupContext(ctx context.Context, obj runtime.Object) context.Context {
	ctx = tracing.Extract(ctx, obj)
	tracing.Inject(ctx, obj)
	return ctx
}

func (h *handler) OnNamespaceRemove(key string, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	logger := h.logger.With("key", key)
	logger.Debug("remove queue")
	return namespace, nil
}

func (h *handler) OnNamespaceChange(key string, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	traceCtx := h.setupContext(context.TODO(), namespace)
	_, span := tracer.Start(traceCtx, "OnNamespaceChange")
	defer span.End()
	logger := h.logger.With("key", key)
	logger.Debug("on change")
	if namespace == nil {
		logger.Debug("namespace is nil")
		return nil, nil
	}
	labels := namespace.Labels
	logger.With("labels", len(labels)).Debug("got labels")
	op, ok := namespace.Labels["controller-tracing-example"]
	if key == "foo" {
		logger.Info("foo key")
	}
	if !ok {
		logger.Info("no label found")
		return namespace, nil
	}
	logger.With("op", op).Debug("got operation")
	if strings.ToLower(op) == "a" {
		logger.Info("operation requested")
		namespace.Labels["controller-tracing-example"] = "B"

		logger.Info("updating namespace...")

		newNamespace, err := h.namespaces.Update(namespace)
		if err != nil {
			logger.Error(err.Error())
			return namespace, err
		}
		return newNamespace, nil
	}
	return namespace, nil
}

func Register(ctx context.Context, app *app.AppContext) {
	logger := slog.Default().With("controller", "A")
	namespaces := app.Core.Namespace()

	h := &handler{
		logger:     logger,
		namespaces: namespaces,
	}

	// namespaces.OnRemove(ctx, "namespace-remove-A", h.OnNamespaceRemove)
	namespaces.OnChange(ctx, "namespace-change-A", h.OnNamespaceChange)
}
