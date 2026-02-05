package models

import "time"

// VideoInfo represents video metadata
type VideoInfo struct {
	VideoID        string     `json:"videoId"`
	VideoURL       string     `json:"videoUrl"`
	UrlType        UrlType    `json:"urlType"`
	DownloadFormat DownloadFormat `json:"downloadFormat"`
}

// UrlType represents the type of video URL
type UrlType int

const (
	UrlTypeOther UrlType = iota
	UrlTypeYouTube
	UrlTypePyPyDance
	UrlTypeVRDancing
)

// DownloadFormat represents the video download format
type DownloadFormat int

const (
	DownloadFormatMP4 DownloadFormat = iota
	DownloadFormatWebm
)

func (f DownloadFormat) String() string {
	switch f {
	case DownloadFormatMP4:
		return "mp4"
	case DownloadFormatWebm:
		return "webm"
	default:
		return "unknown"
	}
}

// CacheEntry represents a cached video file
type CacheEntry struct {
	ID          string    `json:"id"`
	FileName    string    `json:"filename"`
	Size        int64     `json:"size"`
	LastAccess  time.Time `json:"lastAccess"`
	Created     time.Time `json:"created"`
}
