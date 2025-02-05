package volume

type HostMount struct {
	HostPath  string
	MountPath string
}

var (
	Sysfs = HostMount{
		HostPath:  "/opt",
		MountPath: "/host/sys/fs",
	}
)
