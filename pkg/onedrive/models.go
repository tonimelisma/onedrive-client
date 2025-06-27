package onedrive

import "time"

// DriveItemList represents a collection of DriveItems.
type DriveItemList struct {
	Value []DriveItem `json:"value"`
}

// DriveItem represents a file, folder, or other item stored in a drive.
type DriveItem struct {
	CreatedDateTime      time.Time `json:"createdDateTime"`
	CTag                 string    `json:"cTag"`
	ETag                 string    `json:"eTag"`
	ID                   string    `json:"id"`
	LastModifiedDateTime time.Time `json:"lastModifiedDateTime"`
	Name                 string    `json:"name"`
	Size                 int64     `json:"size"`
	WebURL               string    `json:"webUrl"`
	Reactions            struct {
		CommentCount int `json:"commentCount"`
	} `json:"reactions"`
	CreatedBy struct {
		Application struct {
			DisplayName string `json:"displayName"`
			ID          string `json:"id"`
		} `json:"application"`
		User struct {
			DisplayName string `json:"displayName"`
			ID          string `json:"id"`
		} `json:"user"`
	} `json:"createdBy"`
	LastModifiedBy struct {
		Application struct {
			DisplayName string `json:"displayName"`
			ID          string `json:"id"`
		} `json:"application"`
		User struct {
			DisplayName string `json:"displayName"`
			ID          string `json:"id"`
		} `json:"user"`
	} `json:"lastModifiedBy"`
	ParentReference struct {
		DriveID   string `json:"driveId"`
		DriveType string `json:"driveType"`
		ID        string `json:"id"`
		Path      string `json:"path"`
	} `json:"parentReference"`
	FileSystemInfo struct {
		CreatedDateTime      time.Time `json:"createdDateTime"`
		LastModifiedDateTime time.Time `json:"lastModifiedDateTime"`
	} `json:"fileSystemInfo"`
	Folder *struct {
		ChildCount int `json:"childCount"`
		View       struct {
			ViewType  string `json:"viewType"`
			SortBy    string `json:"sortBy"`
			SortOrder string `json:"sortOrder"`
		} `json:"view"`
	} `json:"folder,omitempty"`
	SpecialFolder *struct {
		Name string `json:"name"`
	} `json:"specialFolder,omitempty"`
	RemoteItem *struct {
		ID             string `json:"id"`
		Size           int64  `json:"size"`
		WebURL         string `json:"webUrl"`
		FileSystemInfo struct {
			CreatedDateTime      time.Time `json:"createdDateTime"`
			LastModifiedDateTime time.Time `json:"lastModifiedDateTime"`
		} `json:"fileSystemInfo"`
	} `json:"remoteItem,omitempty"`
}

// UploadSession represents the response from creating an upload session.
type UploadSession struct {
	UploadURL          string   `json:"uploadUrl"`
	ExpirationDateTime string   `json:"expirationDateTime"`
	NextExpectedRanges []string `json:"nextExpectedRanges"`
}

// FolderFacet provides information about the folder metadata of an item.
type FolderFacet struct {
	ChildCount int `json:"childCount"`
}
