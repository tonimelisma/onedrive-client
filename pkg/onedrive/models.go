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
	DownloadURL          string    `json:"@microsoft.graph.downloadUrl"`
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

// DriveList represents a collection of Drives.
type DriveList struct {
	Value []Drive `json:"value"`
}

// Drive represents a drive resource in OneDrive.
type Drive struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	DriveType string `json:"driveType"`
	Owner     struct {
		User struct {
			DisplayName string `json:"displayName"`
		} `json:"user"`
	} `json:"owner"`
	Quota struct {
		Total     int64  `json:"total"`
		Used      int64  `json:"used"`
		Remaining int64  `json:"remaining"`
		State     string `json:"state"`
	} `json:"quota"`
}

// DeviceCodeResponse holds the response from the device code endpoint.
type DeviceCodeResponse struct {
	UserCode        string `json:"user_code"`
	DeviceCode      string `json:"device_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Message         string `json:"message"`
}

// User represents a user resource in OneDrive.
type User struct {
	DisplayName       string `json:"displayName"`
	UserPrincipalName string `json:"userPrincipalName"`
	ID                string `json:"id"`
}

// CopyOperationStatus represents the status of an async copy operation.
type CopyOperationStatus struct {
	Status             string `json:"status"`             // "inProgress", "completed", "failed"
	PercentageComplete int    `json:"percentageComplete"` // 0-100
	StatusDescription  string `json:"statusDescription"`  // Human readable status
	Error              *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	ResourceLocation string `json:"resourceLocation,omitempty"` // URL of completed item
}

// CreateLinkRequest represents the request body for creating a sharing link.
type CreateLinkRequest struct {
	Type     string `json:"type"`               // "view", "edit", "embed"
	Scope    string `json:"scope"`              // "anonymous", "organization"
	Password string `json:"password,omitempty"` // Optional password
}

// SharingLink represents a sharing link returned by the createLink API.
type SharingLink struct {
	ID          string   `json:"id"`
	Roles       []string `json:"roles"`
	ShareId     string   `json:"shareId,omitempty"`
	HasPassword bool     `json:"hasPassword,omitempty"`
	Link        struct {
		Type        string `json:"type"`              // "view", "edit", "embed"
		Scope       string `json:"scope"`             // "anonymous", "organization"
		WebUrl      string `json:"webUrl"`            // The sharing URL
		WebHtml     string `json:"webHtml,omitempty"` // HTML for embedding (embed type only)
		Application struct {
			Id          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"application,omitempty"`
	} `json:"link"`
	ExpirationDateTime string `json:"expirationDateTime,omitempty"`
}

// DeltaResponse represents the response from a delta query
type DeltaResponse struct {
	Value     []DriveItem `json:"value"`
	DeltaLink string      `json:"@odata.deltaLink,omitempty"`
	NextLink  string      `json:"@odata.nextLink,omitempty"`
}

// DriveItemVersion represents a version of a file
type DriveItemVersion struct {
	ID                   string    `json:"id"`
	LastModifiedDateTime time.Time `json:"lastModifiedDateTime"`
	Size                 int64     `json:"size"`
	LastModifiedBy       struct {
		User struct {
			DisplayName string `json:"displayName"`
			ID          string `json:"id"`
		} `json:"user"`
	} `json:"lastModifiedBy"`
}

// DriveItemVersionList represents a collection of file versions
type DriveItemVersionList struct {
	Value []DriveItemVersion `json:"value"`
}
