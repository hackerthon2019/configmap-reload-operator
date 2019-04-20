package appservice

import (
	"context"
	"reflect"
	"strconv"

	cachev1alpha1 "github.com/hackerthon2019/configmap-reload-operator/pkg/apis/app/v1alpha1"
	// kubeCtl "github.com/hackerthon2019/configmap-reload-operator/pkg/kubectl"
	"github.com/hackerthon2019/configmap-reload-operator/pkg/utils"

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
	err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &cachev1alpha1.AppService{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileApp{}
var configMapList [3]*corev1.ConfigMap
var digests []string

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

	// Define a new configmap
	configMapList[0] = r.configmapForApp(app, "nginx-default-conf", "default.conf", utils.NginxConf)
	reqLogger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", configMapList[0].Namespace, "ConfigMap.Name", configMapList[0].Name)
	err = r.client.Create(context.TODO(), configMapList[0])
	if err != nil {
		reqLogger.Error(err, "Failed to create new ConfigMap", "ConfigMap.Namespace", configMapList[0].Namespace, "ConfigMap.Name", configMapList[0].Name)
	}

	// Define a new configmap
	configMapList[1] = r.configmapForApp(app, "nginx-index", "index.html", utils.IndexHTML)
	reqLogger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", configMapList[1].Namespace, "ConfigMap.Name", configMapList[1].Name)
	err = r.client.Create(context.TODO(), configMapList[1])
	if err != nil {
		reqLogger.Error(err, "Failed to create new ConfigMap", "ConfigMap.Namespace", configMapList[1].Namespace, "ConfigMap.Name", configMapList[1].Name)
	}

	// Define a new configmap
	configMapList[2] = r.configmapForApp(app, "nginx-index-dev", "index-dev.html", utils.IndexDevHTML)
	reqLogger.Info("Creating a new ConfigMap", "ConfigMap.Namespace", configMapList[2].Namespace, "ConfigMap.Name", configMapList[2].Name)
	err = r.client.Create(context.TODO(), configMapList[2])
	if err != nil {
		reqLogger.Error(err, "Failed to create new ConfigMap", "ConfigMap.Namespace", configMapList[2].Namespace, "ConfigMap.Name", configMapList[2].Name)
	}

	// Define a new service
	svc := r.serviceForApp(app)
	reqLogger.Info("Creating a new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
	err = r.client.Create(context.TODO(), svc)
	if err != nil {
		reqLogger.Error(err, "Failed to create new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
	}

	for i, cmName := range app.Spec.Dynamic {
		digests = append(digests, utils.ToMD5String(configMapList[i].Data[cmName]))
	}

	// Define a new deployment
	dep := r.deploymentForApp(app, digests)
	reqLogger.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
	err = r.client.Create(context.TODO(), dep)
	if err != nil {
		reqLogger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
	}

	cmFound := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: "nginx-index", Namespace: request.Namespace}, cmFound)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("ConfigMap resource not found. Ignoring since object must be deleted")
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		reqLogger.Error(err, "Failed to get ConfigMap")
		return reconcile.Result{}, err
	}
	// for _, f := range app.Spec.Reload {
	// 	reqLogger.Info("Comparing the " + f + " file...")
	// 	if utils.IsSameMD5(cm.Data[f], cmFound.Data[f]) {
	// 		reqLogger.Info(f + " has same name as previous.")
	// 	} else {
	// 		reqLogger.Info(f + " should now do comfigmap reload!")
	// 	}
	// }

	for k, v := range app.Spec.Selector {
		reqLogger.Info("MyKEY: " + k)
		reqLogger.Info("MyVALUE: " + v)
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
func (r *ReconcileApp) configmapForApp(m *cachev1alpha1.AppService, name string, filename string, content string) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Data: map[string]string{
			filename: content,
		},
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
			Name:      "nginx-expose",
			Namespace: "default",
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
func (r *ReconcileApp) deploymentForApp(m *cachev1alpha1.AppService, digests []string) *appsv1.Deployment {
	ls := labelsForApp(m.Name)
	for i, d := range digests {
		labelName := "configmap-data-md5-" + strconv.Itoa(i)
		ls[labelName] = d
	}
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
								Name:      "nginx-default-conf",
								ReadOnly:  true,
								MountPath: "/etc/nginx/conf.d",
							},
							{
								Name:      "nginx-index",
								ReadOnly:  true,
								MountPath: "/usr/share/nginx/html",
							},
							{
								Name:      "nginx-index-dev",
								ReadOnly:  true,
								MountPath: "/usr/share/nginx/dev/html",
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 80,
							Name:          "nginx-app",
						}},
					}},
					Volumes: []corev1.Volume{
						{
							Name: "nginx-default-conf",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "nginx-default-conf",
									},
								},
							},
						},
						{
							Name: "nginx-index",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "nginx-index",
									},
								},
							},
						},
						{
							Name: "nginx-index-dev",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "nginx-index-dev",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	// Set App instance as the owner and controller
	controllerutil.SetControllerReference(m, dep, r.scheme)
	return dep
}

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
