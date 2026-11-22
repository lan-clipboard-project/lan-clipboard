package handler

import "strings"

func uploaderFromUA(ua string) string {
	ua = strings.ToLower(ua)

	var system string
	switch {
	case strings.Contains(ua, "iphone"), strings.Contains(ua, "ipad"), strings.Contains(ua, "ipod"):
		system = "iOS"
	case strings.Contains(ua, "android"):
		system = "Android"
	case strings.Contains(ua, "windows"):
		system = "Windows"
	case strings.Contains(ua, "cros "):
		system = "ChromeOS"
	case strings.Contains(ua, "macintosh"), strings.Contains(ua, "mac os x"):
		system = "macOS"
	case strings.Contains(ua, "ubuntu"):
		system = "Ubuntu"
	case strings.Contains(ua, "linux"):
		system = "Linux"
	}

	var browser string
	switch {
	case strings.Contains(ua, "edg/"), strings.Contains(ua, "edga/"), strings.Contains(ua, "edgios/"), strings.Contains(ua, "edge/"):
		browser = "Edge"
	case strings.Contains(ua, "opr/"), strings.Contains(ua, "opera"), strings.Contains(ua, "opios/"):
		browser = "Opera"
	case strings.Contains(ua, "samsungbrowser/"):
		browser = "Samsung Internet"
	case strings.Contains(ua, "vivaldi/"):
		browser = "Vivaldi"
	case strings.Contains(ua, "firefox/"), strings.Contains(ua, "fxios/"):
		browser = "Firefox"
	case strings.Contains(ua, "chromium/"):
		browser = "Chromium"
	case strings.Contains(ua, "chrome/"), strings.Contains(ua, "crios/"):
		browser = "Chrome"
	case strings.Contains(ua, "safari/") && strings.Contains(ua, "version/"):
		browser = "Safari"
	}
	return strings.TrimSpace(system + " " + browser)
}
