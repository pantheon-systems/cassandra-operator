package resource

type builderOp struct {
	ServiceType        ClusterServiceType
	PodNumber          int
	ServiceName        string
	ServiceAccountName string
}

// BuilderOption is a function that sets the configuration on the builderOp
type BuilderOption func(*builderOp)

func newBuilderOp() *builderOp {
	op := &builderOp{}
	op.setDefaults()
	return op
}

func (op *builderOp) setDefaults() {}

func (op *builderOp) applyOpts(opts []BuilderOption) {
	for _, opt := range opts {
		opt(op)
	}
}

// WithServiceType sets the service type
func WithServiceType(serviceType ClusterServiceType) BuilderOption {
	return func(op *builderOp) {
		op.ServiceType = serviceType
	}
}

// WithPodNumber sets the pod number
func WithPodNumber(podNumber int) BuilderOption {
	return func(op *builderOp) {
		op.PodNumber = podNumber
	}
}

// WithServiceName sets the service name
func WithServiceName(serviceName string) BuilderOption {
	return func(op *builderOp) {
		op.ServiceName = serviceName
	}
}

// WithServiceAccountName sets the service account name
func WithServiceAccountName(serviceAccountName string) BuilderOption {
	return func(op *builderOp) {
		op.ServiceAccountName = serviceAccountName
	}
}
