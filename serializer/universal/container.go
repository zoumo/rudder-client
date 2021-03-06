package universal

import (
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type Container struct {
	Name               string                      `json:"name"`
	Image              string                      `json:"image"`
	ImagePullPolicy    corev1.PullPolicy           `json:"imagePullPolicy"`
	TTY                bool                        `json:"tty"`
	Command            []string                    `json:"command"`
	Args               []string                    `json:"args"`
	WorkingDir         string                      `json:"workingDir,omitempty"`
	SecurityContext    *corev1.SecurityContext     `json:"securityContext,omitempty"`
	Ports              []corev1.ContainerPort      `json:"ports,omitempty"`
	Env                []corev1.EnvVar             `json:"env,omitempty"`
	EnvFrom            []EnvFrom                   `json:"envFrom,omitempty"`
	Resources          corev1.ResourceRequirements `json:"resources"`
	Mounts             []VolumeMount               `json:"mounts,omitempty"`
	Probe              *ContainerProbe             `json:"probe,omitempty"`
	Lifecycle          *corev1.Lifecycle           `json:"lifecycle,omitempty"`
	ConsoleIsEnvCustom *bool                       `json:"__isEnvCustom,omitempty"`
	ConsoleIsEnvFrom   *bool                       `json:"__isEnvFrom,omitempty"`
	ConsoleIsCommand   *bool                       `json:"__isCommand,omitempty"`
	ConsoleIsMountFile *bool                       `json:"__isMountFile,omitempty"`
	ConsoleIsLog       *bool                       `json:"__isLog,omitempty"`
	ConsoleLiveness    *bool                       `json:"__liveness,omitempty"`
	ConsoleReadiness   *bool                       `json:"__readiness,omitempty"`
}

type EnvFrom struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type VolumeMount struct {
	Name        string `json:"name"`
	ReadOnly    bool   `json:"readonly,omitempty"`
	MountPath   string `json:"path"`
	SubPath     string `json:"subpath,omitempty"`
	ConsoleKind string `json:"__kind,omitempty"`
}

type ContainerProbe struct {
	Liveness  *Probe `json:"liveness,omitempty"`
	Readiness *Probe `json:"readiness,omitempty"`
}

type Probe struct {
	Handler             Handler    `json:"handler,inline"`
	InitialDelaySeconds int32      `json:"delay,omitempty"`
	TimeoutSeconds      int32      `json:"timeout,omitempty"`
	PeriodSeconds       int32      `json:"period,omitempty"`
	Threshold           *Threshold `json:"threshold,omitempty"`
}

type Threshold struct {
	SuccessThreshold int32 `json:"success,omitempty"`
	FailureThreshold int32 `json:"failure,omitempty"`
}

type Handler struct {
	Type   string      `json:"type"`
	Method interface{} `json:"method"`
}

type HTTPGetAction struct {
	Path        string              `json:"path,omitempty"`
	Port        intstr.IntOrString  `json:"port"`
	Host        string              `json:"host,omitempty"`
	Scheme      corev1.URIScheme    `json:"scheme,omitempty"`
	HTTPHeaders []corev1.HTTPHeader `json:"headers,omitempty"`
}

func GetContainers(pod *Pod, containers []corev1.Container, volumes []*Volume) []*Container {
	ret := make([]*Container, 0, len(containers))
	for _, c := range containers {
		vmounts := convertVolumeMounts(c.VolumeMounts, volumes)
		con := &Container{
			Name:               c.Name,
			Image:              c.Image,
			ImagePullPolicy:    c.ImagePullPolicy,
			TTY:                c.TTY,
			Command:            c.Command,
			Args:               c.Args,
			WorkingDir:         c.WorkingDir,
			SecurityContext:    c.SecurityContext,
			Ports:              c.Ports,
			EnvFrom:            convertEnvFrom(c.EnvFrom),
			Env:                c.Env,
			Resources:          c.Resources,
			Mounts:             vmounts,
			Probe:              convertContainerProbe(c.LivenessProbe, c.ReadinessProbe),
			Lifecycle:          c.Lifecycle,
			ConsoleIsEnvCustom: getConsoleIsEnvCustom(&c),
			ConsoleIsEnvFrom:   getConsoleIsEnvFrom(&c),
			ConsoleIsCommand:   getConsoleIsCommand(&c),
			ConsoleIsMountFile: getConsoleIsMountFile(vmounts),
			ConsoleIsLog:       getConsoleIsLog(pod),
			ConsoleLiveness:    getConsoleLiveness(&c),
			ConsoleReadiness:   getConsoleReadiness(&c),
		}
		ret = append(ret, con)
	}
	return ret
}

// =================================================================================================

func convertVolumeMounts(vmounts []corev1.VolumeMount, volumes []*Volume) []VolumeMount {
	vmap := make(map[string]*Volume)
	for c, _ := range volumes {
		vmap[volumes[c].Name] = volumes[c]
	}
	ret := make([]VolumeMount, 0, len(vmounts))
	for _, vmount := range vmounts {
		if v, ok := vmap[vmount.Name]; ok {
			ret = append(ret, VolumeMount{
				Name:        vmount.Name,
				ReadOnly:    vmount.ReadOnly,
				MountPath:   vmount.MountPath,
				SubPath:     vmount.SubPath,
				ConsoleKind: v.ConsoleKind})
		}
	}
	return ret
}

// =================================================================================================

func convertEnvFrom(envFrom []corev1.EnvFromSource) []EnvFrom {
	if len(envFrom) == 0 {
		return nil
	}
	ret := make([]EnvFrom, 0)
	for _, v := range envFrom {
		switch {
		case v.ConfigMapRef != nil:
			ret = append(ret, EnvFrom{Type: "Config", Name: v.ConfigMapRef.Name})
		case v.SecretRef != nil:
			ret = append(ret, EnvFrom{Type: "Secret", Name: v.SecretRef.Name})
		}
	}
	return ret
}

// =================================================================================================

func convertContainerProbe(liveness, readiness *corev1.Probe) *ContainerProbe {
	ret := new(ContainerProbe)
	if liveness != nil {
		ret.Liveness = convertProbe(liveness)
	}
	if readiness != nil {
		ret.Readiness = convertProbe(readiness)
	}
	return ret
}

func convertProbe(probe *corev1.Probe) *Probe {
	return &Probe{
		Handler:             convertHandler(probe.Handler),
		InitialDelaySeconds: probe.InitialDelaySeconds,
		TimeoutSeconds:      probe.TimeoutSeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		Threshold: &Threshold{
			SuccessThreshold: probe.SuccessThreshold,
			FailureThreshold: probe.FailureThreshold,
		},
	}
}

func convertHandler(handler corev1.Handler) Handler {
	ret := Handler{}
	switch {
	case handler.Exec != nil:
		ret.Type = "EXEC"
		ret.Method = handler.Exec
	case handler.HTTPGet != nil:
		ret.Type = "HTTP"
		ret.Method = &HTTPGetAction{
			Path:        handler.HTTPGet.Path,
			Port:        handler.HTTPGet.Port,
			Host:        handler.HTTPGet.Host,
			Scheme:      handler.HTTPGet.Scheme,
			HTTPHeaders: handler.HTTPGet.HTTPHeaders,
		}
	case handler.TCPSocket != nil:
		ret.Type = "TCP"
		ret.Method = handler.TCPSocket
	default:
		glog.Errorf("unsuport handler: %s", handler)
	}

	return ret
}

// =================================================================================================

func getConsoleIsEnvCustom(c *corev1.Container) *bool {
	return convertBoolToPointer(c.Env != nil && len(c.Env) != 0)
}

func getConsoleIsEnvFrom(c *corev1.Container) *bool {
	return convertBoolToPointer(c.EnvFrom != nil && len(c.EnvFrom) != 0)
}

func getConsoleIsCommand(c *corev1.Container) *bool {
	return convertBoolToPointer(c.Command != nil && len(c.Command) != 0)
}

func getConsoleIsLog(pod *Pod) *bool {
	for _, anno := range pod.Annotations {
		if anno.Key == "logging.caicloud.io/required-logfiles" {
			return convertBoolToPointer(true)
		}
	}
	return convertBoolToPointer(false)
}

func getConsoleIsMountFile(vmounts []VolumeMount) *bool {
	for _, vm := range vmounts {
		if vm.ConsoleKind != "" {
			return convertBoolToPointer(true)
		}
	}
	return convertBoolToPointer(false)
}

func getConsoleLiveness(c *corev1.Container) *bool {
	if c == nil {
		return convertBoolToPointer(false)
	}
	if c.LivenessProbe == nil {
		return convertBoolToPointer(false)
	}
	return convertBoolToPointer(true)
}

func getConsoleReadiness(c *corev1.Container) *bool {
	if c == nil {
		return convertBoolToPointer(false)
	}
	if c.ReadinessProbe == nil {
		return convertBoolToPointer(false)
	}
	return convertBoolToPointer(true)
}
