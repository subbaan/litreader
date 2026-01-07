package library

import (
	"os"
	"regexp"
	"strconv"
)

var ratingRegex = regexp.MustCompile(`Average Rating:\s*(\d+\.\d+)`)

// ExtractRating extracts the average rating from a file
// Returns 0.0 if no rating is found
func ExtractRating(filePath string) (float64, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0.0, err
	}

	// Convert to string (UTF-8, ignoring errors)
	content := string(data)

	// Search for rating pattern
	matches := ratingRegex.FindStringSubmatch(content)
	if len(matches) >= 2 {
		rating, err := strconv.ParseFloat(matches[1], 64)
		if err == nil {
			return rating, nil
		}
	}

	return 0.0, nil
}
