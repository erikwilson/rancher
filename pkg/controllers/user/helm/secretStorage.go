package helm

import (
	"github.com/rancher/types/config"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func secretStorageUpgrade(user *config.UserContext) error {
	ownerTiller := labels.Set(map[string]string{"OWNER": "TILLER"})
	configs, err := user.Core.ConfigMaps("").List(metav1.ListOptions{LabelSelector: ownerTiller.String()})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if len(configs.Items) == 0 {
		return nil
	}

	for _, config := range configs.Items {
		namespace := config.ObjectMeta.Namespace
		name := config.ObjectMeta.Name

		configMapClient := user.Core.ConfigMaps(namespace)
		secretClient := user.Core.Secrets(namespace)

		nameApp := labels.Set(map[string]string{"metadata.name": namespace})
		app, err := user.Management.Project.Apps("").List(metav1.ListOptions{FieldSelector: nameApp.String()})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}

		if len(app.Items) == 0 {
			logrus.Warnf("App not found for configMap %s in namespace %s, deleting configMap", name, namespace)
			if err := configMapClient.Delete(name, &metav1.DeleteOptions{}); err != nil {
				return err
			}
			continue
		}

		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: config.ObjectMeta.Labels,
			},
			Data: map[string][]byte{"release": []byte(config.Data["release"])},
		}

		_, err = secretClient.Get(name, metav1.GetOptions{})
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		if apierrors.IsNotFound(err) {
			_, err = secretClient.Create(secret)
		} else {
			_, err = secretClient.Update(secret)
		}
		if err != nil {
			return err
		}

		if err := configMapClient.Delete(name, &metav1.DeleteOptions{}); err != nil {
			return err
		}
	}
	return nil
}
