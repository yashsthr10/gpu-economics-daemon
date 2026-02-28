// Package collector implements telemetry collection from NVML and optional process attribution.
package collector

import (
	"fmt"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/mavvrik/gpu-economics-agent/internal/models"
)

// brandString returns a human-readable label for NVML BrandType (numeric enum).
func brandString(b nvml.BrandType) string {
	switch uint32(b) {
	case 0:
		return "Unknown"
	case 1:
		return "NVIDIA"
	case 2:
		return "Quadro"
	case 3:
		return "Tesla"
	case 4:
		return "GeForce"
	case 5:
		return "Titan"
	case 6:
		return "NVIDIA_VWS"
	case 7:
		return "NVIDIA_VAPPS"
	case 8:
		return "NVIDIA_VPC"
	case 9:
		return "NVIDIA_VCS"
	case 10:
		return "NVIDIA_VGAMING"
	default:
		return fmt.Sprintf("Brand_%d", b)
	}
}

// NVMLReader wraps NVML init, shutdown, and snapshot reading.
// On any NVML error it returns an error; the caller should increment gpu.nvml.errors and retry.
type NVMLReader struct {
	initialized bool
}

// Init initializes the NVML library. Must be called before ReadSnapshot.
func (r *NVMLReader) Init() error {
	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("nvml init: %s", nvml.ErrorString(ret))
	}
	r.initialized = true
	return nil
}

// Shutdown tears down the NVML library. Safe to call if not initialized.
func (r *NVMLReader) Shutdown() error {
	if !r.initialized {
		return nil
	}
	ret := nvml.Shutdown()
	r.initialized = false
	if ret != nvml.SUCCESS {
		return fmt.Errorf("nvml shutdown: %s", nvml.ErrorString(ret))
	}
	return nil
}

// ReadSnapshot enumerates all GPUs and returns one GPUSample per device (spec section 3.1).
// Power and power limit are converted from NVML milliwatts to watts.
// On partial failure (e.g. one device errors), returns a partial slice and no error, or error if no devices read.
func (r *NVMLReader) ReadSnapshot() ([]models.GPUSample, error) {
	if !r.initialized {
		return nil, fmt.Errorf("nvml not initialized")
	}
	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return nil, fmt.Errorf("device count: %s", nvml.ErrorString(ret))
	}
	var out []models.GPUSample
	for i := 0; i < count; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			continue
		}
		gpu := models.GPUSample{GPUIndex: i}

		// Identity and model (same as extract_gpu_snapshot script)
		if name, r := device.GetName(); r == nvml.SUCCESS {
			gpu.Name = name
		}
		if brand, r := device.GetBrand(); r == nvml.SUCCESS {
			gpu.Brand = brandString(brand)
		}
		if uuid, r := device.GetUUID(); r == nvml.SUCCESS {
			gpu.UUID = uuid
		}
		if serial, r := device.GetSerial(); r == nvml.SUCCESS {
			gpu.Serial = serial
		}

		// Power (NVML returns milliwatts)
		if powerMW, r := device.GetPowerUsage(); r == nvml.SUCCESS {
			gpu.PowerWatts = float64(powerMW) / 1000.0
		}
		if limitMW, r := device.GetPowerManagementLimit(); r == nvml.SUCCESS {
			gpu.PowerLimitW = float64(limitMW) / 1000.0
		}
		if minMW, maxMW, r := device.GetPowerManagementLimitConstraints(); r == nvml.SUCCESS {
			gpu.PowerLimitMinW = float64(minMW) / 1000.0
			gpu.PowerLimitMaxW = float64(maxMW) / 1000.0
		}
		if enforcedMW, r := device.GetEnforcedPowerLimit(); r == nvml.SUCCESS {
			gpu.EnforcedPowerLimitW = float64(enforcedMW) / 1000.0
		}

		// Temperature (GPU sensor)
		if temp, r := device.GetTemperature(nvml.TEMPERATURE_GPU); r == nvml.SUCCESS {
			gpu.TemperatureC = float64(temp)
		}

		// Utilization: SM percent and (if no used/total) memory controller percent
		if util, r := device.GetUtilizationRates(); r == nvml.SUCCESS {
			gpu.SMUtilPct = float64(util.Gpu)
			gpu.MemUtilPct = float64(util.Memory)
		}
		// Memory: used/total bytes and recompute utilization from bytes when available
		if mem, r := device.GetMemoryInfo(); r == nvml.SUCCESS {
			gpu.MemoryTotalBytes = mem.Total
			gpu.MemoryUsedBytes = mem.Used
			if mem.Total > 0 {
				gpu.MemUtilPct = 100.0 * float64(mem.Used) / float64(mem.Total)
			}
		}

		// PCIe and compute capability (optional)
		if gen, r := device.GetCurrPcieLinkGeneration(); r == nvml.SUCCESS {
			if width, r2 := device.GetCurrPcieLinkWidth(); r2 == nvml.SUCCESS {
				gpu.PCIeLinkGenWidth = fmt.Sprintf("Gen%d x%d", gen, width)
			}
		}
		if major, minor, r := device.GetCudaComputeCapability(); r == nvml.SUCCESS {
			gpu.ComputeCapability = fmt.Sprintf("%d.%d", major, minor)
		}

		out = append(out, gpu)
	}
	if len(out) == 0 && count > 0 {
		return nil, fmt.Errorf("no GPU metrics could be read")
	}
	return out, nil
}
