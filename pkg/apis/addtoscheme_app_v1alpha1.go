package apis

import (
	"github.com/hackerthon2019/configmap-reload-operator/pkg/apis/app/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme)
}
