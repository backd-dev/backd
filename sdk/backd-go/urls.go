package backd

import "fmt"

// AuthURL returns the auth URL for an app and endpoint
func AuthURL(baseURL, appName, endpoint string) string {
	return fmt.Sprintf("%s/v1/auth/%s/%s", baseURL, appName, endpoint)
}

// DataURL returns the data URL for an app and collection
func DataURL(baseURL, appName, collection string) string {
	return fmt.Sprintf("%s/v1/data/%s/%s", baseURL, appName, collection)
}

// DataItemURL returns the data URL for a specific item
func DataItemURL(baseURL, appName, collection, itemID string) string {
	return fmt.Sprintf("%s/v1/data/%s/%s/%s", baseURL, appName, collection, itemID)
}

// StorageURL returns the storage URL for an app and endpoint
func StorageURL(baseURL, appName, endpoint string) string {
	return fmt.Sprintf("%s/v1/storage/%s/%s", baseURL, appName, endpoint)
}

// StorageFileURL returns the storage URL for a specific file
func StorageFileURL(baseURL, appName, fileID string) string {
	return fmt.Sprintf("%s/v1/storage/%s/files/%s", baseURL, appName, fileID)
}

// FunctionsURL returns the functions URL for an app and function name
func FunctionsURL(baseURL, appName, functionName string) string {
	return fmt.Sprintf("%s/v1/%s/functions/%s", baseURL, appName, functionName)
}
