package cli

var (
	verbose bool

	// all commands
	deviceId string

	// for screenshot command
	screenshotOutputPath  string
	screenshotFormat      string
	screenshotJpegQuality int

	// for screencapture command
	screencaptureFormat string

	// for devices command
	platform   string
	deviceType string

	// for apps launch command
	locale   string
	activity string

	// for agent install command
	agentForce               bool
	agentProvisioningProfile string

	// for fleet allocate command
	fleetType     string
	fleetVersions []string
	fleetNames    []string
	fleetWait     bool
	fleetTimeout  int

	// for webview wait command
	webviewWaitState   string
	webviewWaitTimeout int

	// for EidolonStage integration. Empty means disabled (native only).
	// When set, every device-targeting command first tries to dispatch the
	// operation through the Eidolon MCP server at this endpoint and falls
	// back to native iOS/Android on failure. A value starting with "http" is
	// treated as a URL; any other value is treated as a binary path to spawn
	// over stdio.
	eidolonEndpoint string
)
