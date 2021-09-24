package common

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/logging"
)

// GetLoggingConfig will get config from a specific namespace
func GetLoggingConfig(ctx context.Context, namespace, loggingConfigMapName string) (*logging.Config, error) {
	loggingConfigMap, err := kubeclient.Get(ctx).CoreV1().ConfigMaps(namespace).Get(ctx, loggingConfigMapName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return logging.NewConfigFromMap(nil)
	} else if err != nil {
		return nil, err
	}
	return logging.NewConfigFromConfigMap(loggingConfigMap)
}
