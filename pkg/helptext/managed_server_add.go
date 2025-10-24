package helptext

const (
	ManagedServerAddRootShort = "Add managed servers"

	ManagedServerAddVSphereShort = "Add a VMware vSphere managed server"
	ManagedServerAddVSphereLong  = "Registers a VMware vSphere server (vCenter or ESXi) and optionally provisions credentials automatically."

	ManagedServerAddWindowsShort = "Add a Microsoft Windows managed server"
	ManagedServerAddWindowsLong  = "Registers a Windows server and can create standard credentials automatically or rely on certificate-based authentication when the deployment kit is installed."

	ManagedServerAddLinuxShort = "Add a Linux managed server"
	ManagedServerAddLinuxLong  = "Registers a Linux managed server. Supports permanent credentials, single-use SSH credentials, or certificate-based pairing when the deployment kit is installed."
)
