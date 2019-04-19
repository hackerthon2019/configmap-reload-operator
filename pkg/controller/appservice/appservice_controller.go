package appservice

import (
	"context"
	"reflect"

	cachev1alpha1 "github.com/hackerthon2019/configmap-reload-operator/pkg/apis/app/v1alpha1"
	// kubeCtl "github.com/hackerthon2019/configmap-reload-operator/pkg/kubectl"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_app")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new App Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileApp{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("app-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource App
	err = c.Watch(&source.Kind{Type: &cachev1alpha1.AppService{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner App
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &cachev1alpha1.AppService{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileApp{}

// ReconcileApp reconciles a App object
type ReconcileApp struct {
	// TODO: Clarify the split client
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a App object and makes changes based on the state read
// and what is in the App.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a App Deployment for each App CR
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling App")

	// Fetch the App instance
	app := &cachev1alpha1.AppService{}
	err := r.client.Get(context.TODO(), request.NamespacedName, app)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("App resource not found. Ignoring since object must be deleted")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get App")
		return reconcile.Result{}, err
	}

	for k, v := range app.Spec.Selector {
		reqLogger.Info("MyKEY: " + k)
		reqLogger.Info("MyVALUE: " + v)
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForApp(app)
		reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			reqLogger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return reconcile.Result{}, err
		}

		// Define a new configmap
		cm := r.configmapForApp(app)
		reqLogger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
		err = r.client.Create(context.TODO(), cm)
		if err != nil {
			reqLogger.Error(err, "Failed to create new ConfigMap", "ConfigMap.Namespace", cm.Namespace, "ConfigMap.Name", cm.Name)
			return reconcile.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get Deployment")
		return reconcile.Result{}, err
	}

	// Ensure the deployment size is the same as the spec
	size := app.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.client.Update(context.TODO(), found)
		if err != nil {
			reqLogger.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return reconcile.Result{}, err
		}
		// Spec updated - return and requeue
		return reconcile.Result{Requeue: true}, nil
	}

	// Update the App status with the pod names
	// List the pods for this app's deployment
	podList := &corev1.PodList{}
	labelSelector := labels.SelectorFromSet(labelsForApp(app.Name))
	listOps := &client.ListOptions{Namespace: app.Namespace, LabelSelector: labelSelector}
	err = r.client.List(context.TODO(), listOps, podList)
	if err != nil {
		reqLogger.Error(err, "Failed to list pods", "App.Namespace", app.Namespace, "App.Name", app.Name)
		return reconcile.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, app.Status.Nodes) {
		app.Status.Nodes = podNames
		err := r.client.Status().Update(context.TODO(), app)
		if err != nil {
			reqLogger.Error(err, "Failed to update App status")
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

// configmapForApp returns a app Configmap object
func (r *ReconcileApp) configmapForApp(m *cachev1alpha1.AppService) *corev1.ConfigMap {
	// ls := labelsForApp(m.Name)
	data := map[string]string{"index.html": "<!DOCTYPE html><html><head><meta charset=\"UTF-8\"><title>Title of the document</title></head><body>Content of the document......</body></html>"}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-index",
			Namespace: "default",
		},
		Data: data,
	}
	// Set App instance as the owner and controller
	controllerutil.SetControllerReference(m, cm, r.scheme)
	return cm
}

// serviceForApp returns a app Configmap object
func (r *ReconcileApp) serviceForApp(m *cachev1alpha1.AppService) *corev1.Service {
	ls := labelsForApp(m.Name)
	targetPort := intstr.IntOrString{
		Type:   intstr.Int,
		IntVal: 80,
		StrVal: "80",
	}

	ports := []corev1.ServicePort{
		{
			Name:       "http",
			Port:       int32(80),
			TargetPort: targetPort,
			NodePort:   int32(30080),
		},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx-expose",
		},
		Spec: corev1.ServiceSpec{
			Type:     "NodePort",
			Selector: ls,
			Ports:    ports,
		},
	}
	// Set App instance as the owner and controller
	controllerutil.SetControllerReference(m, svc, r.scheme)
	return svc
}

// deploymentForApp returns a app Deployment object
func (r *ReconcileApp) deploymentForApp(m *cachev1alpha1.AppService) *appsv1.Deployment {
	ls := labelsForApp(m.Name)
	replicas := m.Spec.Size
	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "nginx:stable-alpine",
						Name:  "app",
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "nginx-index",
								ReadOnly:  true,
								MountPath: "/usr/share/nginx/html",
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 80,
							Name:          "nginx-app",
						}},
					}},
					Volumes: []corev1.Volume{{
						Name: "nginx-index",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "nginx-index",
								},
							},
						},
					}},
				},
			},
		},
	}
	// Set App instance as the owner and controller
	controllerutil.SetControllerReference(m, dep, r.scheme)
	return dep
}

// func generateConfigMap(deploy *entity.Deployment) ([]corev1.Volume, []corev1.VolumeMount, error) {
// 	volumes := []corev1.Volume{}
// 	volumeMounts := []corev1.VolumeMount{}
//
// 	for _, v := range deploy.ConfigMaps {
//
// 		// TODO: check whether this configMap exist
//
// 		vName := fmt.Sprintf("%s-%s", ConfigMapNamePrefix, v.Name)
//
// 		volumes = append(volumes, corev1.Volume{
// 			Name: vName,
// 			VolumeSource: corev1.VolumeSource{
// 				ConfigMap: &corev1.ConfigMapVolumeSource{
// 					LocalObjectReference: corev1.LocalObjectReference{
// 						Name: v.Name,
// 					},
// 				},
// 			},
// 		})
//
// 		volumeMounts = append(volumeMounts, corev1.VolumeMount{
// 			Name:      vName,
// 			MountPath: v.MountPath,
// 		})
// 	}
//
// 	return volumes, volumeMounts, nil
// }

// labelsForApp returns the labels for selecting the resources
// belonging to the given app CR name.
func labelsForApp(name string) map[string]string {
	return map[string]string{"app": "app", "app_cr": name}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}
