package chain

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alexandreLamarre/controller-tracing-example/pkg/app"
	"github.com/rancher/lasso/pkg/tracing"
	v1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/core/v1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	corev1 "k8s.io/api/core/v1"
)

var (
	tracer = otel.GetTracerProvider().Tracer("chainNamespaceOperator")
)

type handler struct {
	logger *slog.Logger

	namespaces v1.NamespaceController
}

func (h *handler) CommitChanges(key string, ns *corev1.Namespace) (*corev1.Namespace, error) {
	ctx := tracing.Extract(context.Background(), ns)
	spanCtx, span := tracer.Start(ctx, "CommitChanges")
	defer span.End()
	logger := h.logger.With("key", key)
	logger.Debug("commit changes")
	if ns == nil {
		logger.Debug("namespace is nil")
		return ns, nil
	}

	span.SetStatus(codes.Ok, "namespace committed")
	tracing.Inject(spanCtx, ns)
	return h.namespaces.Update(ns)
}

type labelMutator func(ctx context.Context, namespace *corev1.Namespace) error

// AddLabelMutator returns a function that adds a label to a namespace if it doesn't exist
func AddLabelMutator(key, val string) labelMutator {
	return func(ctx context.Context, namespace *corev1.Namespace) error {
		_, span := tracer.Start(ctx, "AppLogic.AddLabelMutator")
		defer span.End()
		span.SetAttributes(
			attribute.String("target.key", key),
			attribute.String("target.value", val),
		)
		if namespace.Labels == nil {
			namespace.Labels = map[string]string{}
		}
		span.SetStatus(codes.Ok, "label added")
		if _, ok := namespace.Labels[key]; ok {
			return nil
		}
		namespace.Labels[key] = val
		return nil
	}
}

// DeleteLabelMutator returns a function that deletes a label from a namespace, and returns an error if the label is not found
func DeleteLabelMutator(key string) labelMutator {
	return func(ctx context.Context, namespace *corev1.Namespace) error {
		_, span := tracer.Start(ctx, "AppLogic.DeleteLabelMutator")
		defer span.End()
		span.SetAttributes(
			attribute.String("target.key", key),
		)
		if _, ok := namespace.Labels[key]; !ok {
			err := fmt.Errorf("label %s not found", key)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		delete(namespace.Labels, key)
		span.SetStatus(codes.Ok, "label deleted")
		return nil
	}
}

// UpdateLabelMutator returns a function that updates a label in a namespace, and returns an error if the label is not found
func UpdateLabelMutator(key, val string) labelMutator {
	return func(ctx context.Context, namespace *corev1.Namespace) error {
		_, span := tracer.Start(ctx, "AppLogic.UpdateLabelMutator")
		defer span.End()
		span.SetAttributes(
			attribute.String("target.key", key),
			attribute.String("target.value", val),
		)
		oldVal, ok := namespace.Labels[key]
		if !ok {
			err := fmt.Errorf("label %s not found", key)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		span.SetAttributes(
			attribute.String("existing.value", oldVal),
		)
		namespace.Labels[key] = val
		span.SetStatus(codes.Ok, "label updated")
		return nil
	}
}

func (h *handler) OnChangeFactory(name string, mut labelMutator) func(string, *corev1.Namespace) (*corev1.Namespace, error) {
	return func(key string, namespace *corev1.Namespace) (*corev1.Namespace, error) {
		ctx := tracing.Extract(context.Background(), namespace)
		spanCtx, span := tracer.Start(ctx, fmt.Sprintf("OnNamespaceChange/%s", name))
		defer span.End()
		logger := h.logger.With("key", key)
		logger.Debug("on change")
		if namespace == nil {
			logger.Debug("namespace is nil")
			return namespace, nil
		}
		if err := mut(spanCtx, namespace); err != nil {
			logger.Error("failed to mutate label", "error", err)
			span.SetStatus(codes.Error, err.Error())
			return namespace, err
		}
		span.SetStatus(codes.Ok, "label mutator successful")
		return namespace, nil
	}
}

func Register(ctx context.Context, app *app.AppContext) {
	logger := slog.Default().With("controller", "chainNamespace")

	keyF := func(suffix string) string {
		return fmt.Sprintf("%s.%s", "chainNamespace", suffix)
	}

	mutators := []labelMutator{
		AddLabelMutator(keyF("foo"), "bar"),
		UpdateLabelMutator(keyF("foo"), "baz"),
		DeleteLabelMutator(keyF("foo")),
		AddLabelMutator(keyF("bar"), "baz"),
		UpdateLabelMutator(keyF("bar"), "foo"),
		DeleteLabelMutator(keyF("bar")),
		AddLabelMutator(keyF("baz"), "foo"),
		AddLabelMutator(keyF("bar"), "baz"),
		AddLabelMutator(keyF("foo"), "bar"),
	}

	h := &handler{
		logger:     logger,
		namespaces: app.Core.Namespace(),
	}

	for i, mut := range mutators {
		h.namespaces.OnChange(ctx, fmt.Sprintf("chainNamespace-%d", i), h.OnChangeFactory(fmt.Sprintf("chainNamespace-%d", i), mut))
	}
	h.namespaces.OnChange(ctx, "chainNamespace-CommitChanges", h.CommitChanges)
}
