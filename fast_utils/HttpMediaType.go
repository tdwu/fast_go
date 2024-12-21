package fast_utils

import "path/filepath"

// GetFileMediaType 根据文件后缀获取媒体类型
func GetFileMediaType(filename string) string {
	//https://developer.mozilla.org/zh-CN/docs/Web/HTTP/Basics_of_HTTP/MIME_types/Common_types
	// 文件后缀到媒体类型的映射
	extensionMap := map[string]string{
		".webp": "image/webp",
		".html": "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".bmp":  "image/bmp",
		".ico":  "image/x-icon",
		".svg":  "image/svg+xml",
		".mp3":  "audio/mpeg",
		".wav":  "audio/wav",
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".wmv":  "video/x-ms-wmv",
		".flv":  "video/x-flv",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",
		".zip":  "application/zip",
		".rar":  "application/x-rar-compressed",
		".tar":  "application/x-tar",
		".gz":   "application/gzip",
		".7z":   "application/x-7z-compressed",
		".csv":  "text/csv",
		".txt":  "text/plain",
		// 添加其他文件类型的映射
	}

	// 获取文件后缀
	ext := filepath.Ext(filename)

	// 查找对应的媒体类型
	mediaType, found := extensionMap[ext]
	if !found {
		return "application/octet-stream" // 默认媒体类型
	}
	return mediaType
}
