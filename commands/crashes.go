package commands

import "fmt"

func CrashesListCommand(deviceID string) *CommandResponse {
	if resp, handled := TryEidolonDispatch("device.crashes.list", map[string]any{
		"deviceId": deviceID,
	}); handled {
		return resp
	}

	device, err := FindDeviceOrAutoSelect(deviceID)
	if err != nil {
		return NewErrorResponse(fmt.Errorf("error finding device: %w", err))
	}

	crashes, err := device.ListCrashReports()
	if err != nil {
		return NewErrorResponse(fmt.Errorf("error listing crash reports: %w", err))
	}

	return NewSuccessResponse(crashes)
}

func CrashesGetCommand(deviceID string, id string) *CommandResponse {
	if resp, handled := TryEidolonDispatch("device.crashes.get", map[string]any{
		"id":       id,
		"deviceId": deviceID,
	}); handled {
		return resp
	}

	device, err := FindDeviceOrAutoSelect(deviceID)
	if err != nil {
		return NewErrorResponse(fmt.Errorf("error finding device: %w", err))
	}

	content, err := device.GetCrashReport(id)
	if err != nil {
		return NewErrorResponse(fmt.Errorf("error getting crash report: %w", err))
	}

	return NewSuccessResponse(map[string]string{
		"id":      id,
		"content": string(content),
	})
}
