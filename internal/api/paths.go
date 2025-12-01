// Package api provides the Gemini Web API client implementation.
package api

// GJSON paths for extracting values from Gemini responses.
// These centralize the "magic indices" from the Python implementation.
const (
	// Response body paths
	PathBody      = "2"
	PathCandList  = "4"
	PathMetadata  = "1"
	PathErrorCode = "0.5.2.0.1.0"

	// Candidate paths (relative to candidate object)
	PathCandRCID      = "0"
	PathCandText      = "1.0"
	PathCandTextAlt   = "22.0"
	PathCandThoughts  = "37.0.0"
	PathCandWebImages = "12.1"
	PathCandGenImages = "12.7.0"

	// Web image paths (relative to web image object)
	PathWebImgURL   = "0.0.0"
	PathWebImgTitle = "7.0"
	PathWebImgAlt   = "0.4"

	// Generated image paths (relative to generated image object)
	PathGenImgURL  = "0.3.3"
	PathGenImgNum  = "3.6"
	PathGenImgAlts = "3.5"
)
