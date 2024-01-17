package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	composetypes "github.com/compose-spec/compose-go/types"
	"github.com/docker/docker/api/types/blkiodev"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
)

// toMobyEnv converts a compose.MappingWithEquals to a []string
func toMobyEnv(environment composetypes.MappingWithEquals) []string {
	var env []string
	for k, v := range environment {
		if v == nil {
			env = append(env, k)
		} else {
			env = append(env, fmt.Sprintf("%s=%s", k, *v))
		}
	}
	return env
}

func buildContainerPorts(s composetypes.ServiceConfig) nat.PortSet {
	ports := nat.PortSet{}
	for _, s := range s.Expose {
		p := nat.Port(s)
		ports[p] = struct{}{}
	}
	return ports
}

func parseSecurityOpts(p *composetypes.Project, securityOpts []string) ([]string, bool, error) {
	var (
		unconfined bool
		parsed     []string
	)
	for _, opt := range securityOpts {
		if opt == "systempaths=unconfined" {
			unconfined = true
			continue
		}
		con := strings.SplitN(opt, "=", 2)
		if len(con) == 1 && con[0] != "no-new-privileges" {
			if strings.Contains(opt, ":") {
				con = strings.SplitN(opt, ":", 2)
			} else {
				return securityOpts, false, fmt.Errorf("invalid security-opt: %q", opt)
			}
		}
		if con[0] == "seccomp" && con[1] != "unconfined" {
			f, err := os.ReadFile(p.RelativePath(con[1]))
			if err != nil {
				return securityOpts, false, fmt.Errorf("opening seccomp profile (%s) failed: %w", con[1], err)
			}
			b := bytes.NewBuffer(nil)
			if err := json.Compact(b, f); err != nil {
				return securityOpts, false, fmt.Errorf("compacting json for seccomp profile (%s) failed: %w", con[1], err)
			}
			parsed = append(parsed, fmt.Sprintf("seccomp=%s", b.Bytes()))
		} else {
			parsed = append(parsed, opt)
		}
	}

	return parsed, unconfined, nil
}

func getRestartPolicy(service composetypes.ServiceConfig) container.RestartPolicy {
	var restart container.RestartPolicy
	if service.Restart != "" {
		split := strings.Split(service.Restart, ":")
		var attempts int
		if len(split) > 1 {
			attempts, _ = strconv.Atoi(split[1])
		}
		restart = container.RestartPolicy{
			Name:              split[0],
			MaximumRetryCount: attempts,
		}
	}
	if service.Deploy != nil && service.Deploy.RestartPolicy != nil {
		policy := *service.Deploy.RestartPolicy
		var attempts int
		if policy.MaxAttempts != nil {
			attempts = int(*policy.MaxAttempts)
		}
		restart = container.RestartPolicy{
			Name:              policy.Condition,
			MaximumRetryCount: attempts,
		}
	}
	return restart
}

func getDeployResources(s composetypes.ServiceConfig) container.Resources {
	var swappiness *int64
	if s.MemSwappiness != 0 {
		val := int64(s.MemSwappiness)
		swappiness = &val
	}
	resources := container.Resources{
		CgroupParent:       s.CgroupParent,
		Memory:             int64(s.MemLimit),
		MemorySwap:         int64(s.MemSwapLimit),
		MemorySwappiness:   swappiness,
		MemoryReservation:  int64(s.MemReservation),
		OomKillDisable:     &s.OomKillDisable,
		CPUCount:           s.CPUCount,
		CPUPeriod:          s.CPUPeriod,
		CPUQuota:           s.CPUQuota,
		CPURealtimePeriod:  s.CPURTPeriod,
		CPURealtimeRuntime: s.CPURTRuntime,
		CPUShares:          s.CPUShares,
		NanoCPUs:           int64(s.CPUS * 1e9),
		CPUPercent:         int64(s.CPUPercent * 100),
		CpusetCpus:         s.CPUSet,
		DeviceCgroupRules:  s.DeviceCgroupRules,
	}

	if s.PidsLimit != 0 {
		resources.PidsLimit = &s.PidsLimit
	}

	setBlkio(s.BlkioConfig, &resources)

	if s.Deploy != nil {
		setLimits(s.Deploy.Resources.Limits, &resources)
		setReservations(s.Deploy.Resources.Reservations, &resources)
	}

	for _, device := range s.Devices {
		// FIXME should use docker/cli parseDevice, unfortunately private
		src := ""
		dst := ""
		permissions := "rwm"
		arr := strings.Split(device, ":")
		switch len(arr) {
		case 3:
			permissions = arr[2]
			fallthrough
		case 2:
			dst = arr[1]
			fallthrough
		case 1:
			src = arr[0]
		}
		if dst == "" {
			dst = src
		}
		resources.Devices = append(resources.Devices, container.DeviceMapping{
			PathOnHost:        src,
			PathInContainer:   dst,
			CgroupPermissions: permissions,
		})
	}

	ulimits := toUlimits(s.Ulimits)
	resources.Ulimits = ulimits
	return resources
}

func toUlimits(m map[string]*composetypes.UlimitsConfig) []*units.Ulimit {
	var ulimits []*units.Ulimit
	for name, u := range m {
		soft := u.Single
		if u.Soft != 0 {
			soft = u.Soft
		}
		hard := u.Single
		if u.Hard != 0 {
			hard = u.Hard
		}
		ulimits = append(ulimits, &units.Ulimit{
			Name: name,
			Hard: int64(hard),
			Soft: int64(soft),
		})
	}
	return ulimits
}

func setReservations(reservations *composetypes.Resource, resources *container.Resources) {
	if reservations == nil {
		return
	}
	// Cpu reservation is a swarm option and PIDs is only a limit
	// So we only need to map memory reservation and devices
	if reservations.MemoryBytes != 0 {
		resources.MemoryReservation = int64(reservations.MemoryBytes)
	}

	for _, device := range reservations.Devices {
		resources.DeviceRequests = append(resources.DeviceRequests, container.DeviceRequest{
			Capabilities: [][]string{device.Capabilities},
			Count:        int(device.Count),
			DeviceIDs:    device.IDs,
			Driver:       device.Driver,
		})
	}
}

func setLimits(limits *composetypes.Resource, resources *container.Resources) {
	if limits == nil {
		return
	}
	if limits.MemoryBytes != 0 {
		resources.Memory = int64(limits.MemoryBytes)
	}
	if limits.NanoCPUs != "" {
		if f, err := strconv.ParseFloat(limits.NanoCPUs, 64); err == nil {
			resources.NanoCPUs = int64(f * 1e9)
		}
	}
	if limits.Pids > 0 {
		resources.PidsLimit = &limits.Pids
	}
}

func setBlkio(blkio *composetypes.BlkioConfig, resources *container.Resources) {
	if blkio == nil {
		return
	}
	resources.BlkioWeight = blkio.Weight
	for _, b := range blkio.WeightDevice {
		resources.BlkioWeightDevice = append(resources.BlkioWeightDevice, &blkiodev.WeightDevice{
			Path:   b.Path,
			Weight: b.Weight,
		})
	}
	for _, b := range blkio.DeviceReadBps {
		resources.BlkioDeviceReadBps = append(resources.BlkioDeviceReadBps, &blkiodev.ThrottleDevice{
			Path: b.Path,
			Rate: uint64(b.Rate),
		})
	}
	for _, b := range blkio.DeviceReadIOps {
		resources.BlkioDeviceReadIOps = append(resources.BlkioDeviceReadIOps, &blkiodev.ThrottleDevice{
			Path: b.Path,
			Rate: uint64(b.Rate),
		})
	}
	for _, b := range blkio.DeviceWriteBps {
		resources.BlkioDeviceWriteBps = append(resources.BlkioDeviceWriteBps, &blkiodev.ThrottleDevice{
			Path: b.Path,
			Rate: uint64(b.Rate),
		})
	}
	for _, b := range blkio.DeviceWriteIOps {
		resources.BlkioDeviceWriteIOps = append(resources.BlkioDeviceWriteIOps, &blkiodev.ThrottleDevice{
			Path: b.Path,
			Rate: uint64(b.Rate),
		})
	}
}
