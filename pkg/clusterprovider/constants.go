package clusterprovider

const (
	openControlPlaneGroupName = "open-control-plane.io"

	// LocalAccessAnnotation is used for local cluster access when developing providers
	LocalAccessAnnotation = "clusters." + openControlPlaneGroupName + "/local-access"
)
