package library

import (
	"github.com/subbass/litreader/internal/models"
)

// Stats holds library statistics
type Stats struct {
	TotalFiles  int
	TotalSizeMB float64
	AvgRating   float64
	RatedCount  int
}

// CalculateStats computes statistics from file info
func CalculateStats(fileInfo map[string]*models.FileInfo) Stats {
	stats := Stats{}

	if len(fileInfo) == 0 {
		return stats
	}

	var totalSize int64
	var totalRating float64
	var ratedCount int

	for _, info := range fileInfo {
		totalSize += info.Size
		if info.Rating > 0 {
			totalRating += info.Rating
			ratedCount++
		}
	}

	stats.TotalFiles = len(fileInfo)
	stats.TotalSizeMB = float64(totalSize) / (1024 * 1024)
	stats.RatedCount = ratedCount

	if ratedCount > 0 {
		stats.AvgRating = totalRating / float64(ratedCount)
	}

	return stats
}
