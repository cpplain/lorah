package loop

import "time"

const (
	colorReset = "\033[0m"
	colorGreen = "\033[32m"
	colorBlue  = "\033[34m"
	colorBold  = "\033[1m"
	colorRed   = "\033[31m"

	maxBufferSize = 1024 * 1024 // 1MB buffer for JSON parsing
	retryDelay    = 5 * time.Second
)
