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

// Paging represents pagination options for API calls
type Paging struct {
	Top      int    `json:"top,omitempty"`      // 0 = Graph default
	FetchAll bool   `json:"fetchAll,omitempty"` // true → ignore Top and NextLink, loop to end
	NextLink string `json:"nextLink,omitempty"` // if non-empty start exactly from this URL
}

// Activity represents an activity that took place on an item or within a container
type Activity struct {
	ID     string `json:"id"`
	Action struct {
		Comment *struct{} `json:"comment,omitempty"`
		Create  *struct{} `json:"create,omitempty"`
		Delete  *struct{} `json:"delete,omitempty"`
		Edit    *struct {
			NewVersion string `json:"newVersion,omitempty"`
		} `json:"edit,omitempty"`
		Mention *struct {
			Mentionees []struct {
				User struct {
					DisplayName string `json:"displayName"`
				} `json:"user"`
			} `json:"mentionees,omitempty"`
		} `json:"mention,omitempty"`
		Move   *struct{} `json:"move,omitempty"`
		Rename *struct {
			OldName string `json:"oldName,omitempty"`
		} `json:"rename,omitempty"`
		Restore *struct{} `json:"restore,omitempty"`
		Share   *struct{} `json:"share,omitempty"`
		Version *struct {
			NewVersion string `json:"newVersion,omitempty"`
		} `json:"version,omitempty"`
	} `json:"action"`
	Actor struct {
		User struct {
			DisplayName string `json:"displayName"`
			ID          string `json:"id"`
		} `json:"user"`
	} `json:"actor"`
	Times struct {
		RecordedTime time.Time `json:"recordedTime"`
	} `json:"times"`
	DriveItem *DriveItem `json:"driveItem,omitempty"`
}

// ActivityList represents a collection of activities with pagination
type ActivityList struct {
	Value    []Activity `json:"value"`
	NextLink string     `json:"@odata.nextLink,omitempty"`
}

// Thumbnail represents a thumbnail image with size information
type Thumbnail struct {
	Height int    `json:"height"`
	Width  int    `json:"width"`
	URL    string `json:"url"`
}

// ThumbnailSet represents a collection of thumbnail sizes for an item
type ThumbnailSet struct {
	ID     string     `json:"id"`
	Small  *Thumbnail `json:"small,omitempty"`
	Medium *Thumbnail `json:"medium,omitempty"`
	Large  *Thumbnail `json:"large,omitempty"`
	Source *Thumbnail `json:"source,omitempty"`
}

// ThumbnailSetList represents a collection of thumbnail sets
type ThumbnailSetList struct {
	Value []ThumbnailSet `json:"value"`
}

// PreviewRequest represents the request body for item preview
type PreviewRequest struct {
	Page string  `json:"page,omitempty"` // Page number or name to preview
	Zoom float64 `json:"zoom,omitempty"` // Zoom level (1.0 = 100%)
}

// PreviewResponse represents the response from the preview API
type PreviewResponse struct {
	GetURL         string `json:"getUrl,omitempty"`         // URL for GET request
	PostURL        string `json:"postUrl,omitempty"`        // URL for POST request
	PostParameters string `json:"postParameters,omitempty"` // Parameters for POST request
}

// Permission represents a sharing permission on an item
type Permission struct {
	ID                 string   `json:"id"`
	Roles              []string `json:"roles"`                        // "read", "write", "owner"
	ShareID            string   `json:"shareId,omitempty"`            // Share identifier
	ExpirationDateTime string   `json:"expirationDateTime,omitempty"` // ISO 8601 format
	HasPassword        bool     `json:"hasPassword,omitempty"`        // Whether password protected
	PermissionScope    string   `json:"scope,omitempty"`              // "anonymous", "organization", "users"
	GrantedToV2        *struct {
		User *struct {
			DisplayName string `json:"displayName"`
			ID          string `json:"id"`
			Email       string `json:"email,omitempty"`
		} `json:"user,omitempty"`
		SiteUser *struct {
			DisplayName string `json:"displayName"`
			ID          string `json:"id"`
			LoginName   string `json:"loginName,omitempty"`
			Email       string `json:"email,omitempty"`
		} `json:"siteUser,omitempty"`
	} `json:"grantedToV2,omitempty"`
	GrantedToIdentitiesV2 []struct {
		User *struct {
			DisplayName string `json:"displayName"`
			ID          string `json:"id"`
			Email       string `json:"email,omitempty"`
		} `json:"user,omitempty"`
		SiteUser *struct {
			DisplayName string `json:"displayName"`
			ID          string `json:"id"`
			LoginName   string `json:"loginName,omitempty"`
			Email       string `json:"email,omitempty"`
		} `json:"siteUser,omitempty"`
	} `json:"grantedToIdentitiesV2,omitempty"`
	Link *struct {
		Type        string `json:"type"`              // "view", "edit", "embed"
		Scope       string `json:"scope"`             // "anonymous", "organization"
		WebURL      string `json:"webUrl"`            // The sharing URL
		WebHTML     string `json:"webHtml,omitempty"` // HTML for embedding
		Application struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"application,omitempty"`
		PreventsDownload bool `json:"preventsDownload,omitempty"` // Whether download is prevented
	} `json:"link,omitempty"`
	InheritedFrom *struct {
		DriveID string `json:"driveId"`
		ID      string `json:"id"`
		Path    string `json:"path"`
	} `json:"inheritedFrom,omitempty"`
}

// PermissionList represents a collection of permissions
type PermissionList struct {
	Value []Permission `json:"value"`
}

// InviteRequest represents the request body for inviting users
type InviteRequest struct {
	Recipients []struct {
		Email string `json:"email"`
	} `json:"recipients"`
	Message              string   `json:"message,omitempty"`              // Optional invitation message
	RequireSignIn        bool     `json:"requireSignIn,omitempty"`        // Whether sign-in is required
	SendInvitation       bool     `json:"sendInvitation,omitempty"`       // Whether to send email invitation
	Roles                []string `json:"roles"`                          // "read", "write"
	ExpirationDateTime   string   `json:"expirationDateTime,omitempty"`   // ISO 8601 format
	Password             string   `json:"password,omitempty"`             // Optional password
	RetainInheritedRoles bool     `json:"retainInheritedRoles,omitempty"` // Whether to retain inherited permissions
}

// InviteResponse represents the response from inviting users
type InviteResponse struct {
	Value []Permission `json:"value"` // Created permissions for invited users
}

// UpdatePermissionRequest represents the request body for updating permissions
type UpdatePermissionRequest struct {
	Roles              []string `json:"roles,omitempty"`              // "read", "write", "owner"
	ExpirationDateTime string   `json:"expirationDateTime,omitempty"` // ISO 8601 format
	Password           string   `json:"password,omitempty"`           // Optional password
}
