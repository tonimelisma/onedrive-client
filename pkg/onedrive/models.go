// Package onedrive (models.go) defines the various data structures (structs)
// used by the SDK to represent resources and request/response bodies for the
// Microsoft Graph API for OneDrive. These models are used for JSON serialization
// and deserialization when communicating with the API.
package onedrive

import "time"

// DriveItemList represents a collection of DriveItem resources.
// It's commonly used in responses for listing children of a folder or search results.
type DriveItemList struct {
	Value    []DriveItem `json:"value"`                     // The array of DriveItem resources.
	NextLink string      `json:"@odata.nextLink,omitempty"` // URL to the next page of results, if any.
}

// DriveItem represents a file, folder, or other item stored in a OneDrive drive.
// It contains metadata about the item, such as its name, size, timestamps,
// and information about its type (e.g., folder or file specific facets).
type DriveItem struct {
	CreatedDateTime      time.Time `json:"createdDateTime"`      // Timestamp of when the item was created.
	CTag                 string    `json:"cTag"`                 // An eTag for content changes, stays the same if only metadata changes.
	ETag                 string    `json:"eTag"`                 // An eTag for metadata changes.
	ID                   string    `json:"id"`                   // The unique identifier of the DriveItem.
	LastModifiedDateTime time.Time `json:"lastModifiedDateTime"` // Timestamp of when the item was last modified.
	Name                 string    `json:"name"`                 // The name of the DriveItem (e.g., "MyFile.docx").
	Size                 int64     `json:"size"`                 // Size of the item in bytes.
	WebURL               string    `json:"webUrl"`               // URL that displays the item in OneDrive on the web.
	// DownloadURL is a pre-authenticated URL for accessing the item's content.
	// Note the specific JSON tag name used by Microsoft Graph API.
	DownloadURL string `json:"@microsoft.graph.downloadUrl,omitempty"`
	Reactions   struct {
		CommentCount int `json:"commentCount"` // Count of comments associated with the item.
	} `json:"reactions"`
	CreatedBy struct { // Information about the user or application that created the item.
		Application *Identity `json:"application,omitempty"`
		User        *Identity `json:"user,omitempty"`
		Device      *Identity `json:"device,omitempty"` // Added for completeness
	} `json:"createdBy"`
	LastModifiedBy struct { // Information about the user or application that last modified the item.
		Application *Identity `json:"application,omitempty"`
		User        *Identity `json:"user,omitempty"`
		Device      *Identity `json:"device,omitempty"` // Added for completeness
	} `json:"lastModifiedBy"`
	ParentReference struct { // Information about the parent folder of this item.
		DriveID   string `json:"driveId"`   // ID of the drive the parent folder is in.
		DriveType string `json:"driveType"` // Type of the drive (e.g., "personal", "business").
		ID        string `json:"id"`        // ID of the parent folder.
		Path      string `json:"path"`      // Path of the parent folder, relative to the drive root.
	} `json:"parentReference"`
	FileSystemInfo struct { // File system specific metadata.
		CreatedDateTime      time.Time `json:"createdDateTime"`      // Timestamp from the file system when the item was created.
		LastModifiedDateTime time.Time `json:"lastModifiedDateTime"` // Timestamp from the file system when the item was last modified.
	} `json:"fileSystemInfo"`
	Folder        *FolderFacet  `json:"folder,omitempty"`  // If the item is a folder, this contains folder-specific metadata.
	File          *FileFacet    `json:"file,omitempty"`    // If the item is a file, this contains file-specific metadata.
	Image         *ImageFacet   `json:"image,omitempty"`   // If the item is an image, this contains image-specific metadata.
	Video         *VideoFacet   `json:"video,omitempty"`   // If the item is a video, this contains video-specific metadata.
	Audio         *AudioFacet   `json:"audio,omitempty"`   // If the item is an audio file, this contains audio-specific metadata.
	Photo         *PhotoFacet   `json:"photo,omitempty"`   // If the item is a photo, this contains photo-specific metadata.
	Package       *PackageFacet `json:"package,omitempty"` // If the item is a package (e.g., OneNote notebook).
	SpecialFolder *struct {     // If the item is a special folder (e.g., Documents, Photos).
		Name string `json:"name"` // Name of the special folder (e.g., "documents").
	} `json:"specialFolder,omitempty"`
	RemoteItem *RemoteItemFacet `json:"remoteItem,omitempty"` // If the item is a link to an item on another drive.
	Deleted    *DeletedFacet    `json:"deleted,omitempty"`    // If the item has been deleted.
}

// Identity represents an identity of an actor (user, application, or device).
type Identity struct {
	DisplayName string `json:"displayName"` // The display name of the identity.
	ID          string `json:"id"`          // The unique identifier of the identity.
}

// FolderFacet provides metadata specific to items that are folders.
type FolderFacet struct {
	ChildCount int       `json:"childCount"` // Number of children items directly under this folder.
	View       *struct { // Optional: View preferences for the folder.
		ViewType  string `json:"viewType"`  // Type of view (e.g., "thumbnails", "details").
		SortBy    string `json:"sortBy"`    // Field to sort by (e.g., "name", "lastModifiedDateTime").
		SortOrder string `json:"sortOrder"` // Sort order ("ascending" or "descending").
	} `json:"view,omitempty"`
}

// FileFacet provides metadata specific to items that are files.
type FileFacet struct {
	MimeType string    `json:"mimeType"` // The MIME type for the file.
	Hashes   *struct { // Hashes of the file content.
		Sha1Hash   string `json:"sha1Hash,omitempty"`   // SHA1 hash for the contents of the file (if available).
		Sha256Hash string `json:"sha256Hash,omitempty"` // SHA256 hash for the contents of the file (if available).
		Crc32Hash  string `json:"crc32Hash,omitempty"`  // CRC32 hash for the contents of the file (if available).
	} `json:"hashes,omitempty"`
}

// ImageFacet provides metadata specific to items that are images.
type ImageFacet struct {
	Width  int `json:"width"`  // Width of the image, in pixels.
	Height int `json:"height"` // Height of the image, in pixels.
}

// VideoFacet provides metadata specific to items that are videos.
type VideoFacet struct {
	Bitrate    int    `json:"bitrate"`              // Video bitrate in bits per second.
	Duration   int64  `json:"duration"`             // Duration of the video in milliseconds.
	Height     int    `json:"height"`               // Video height in pixels.
	Width      int    `json:"width"`                // Video width in pixels.
	AudioCodec string `json:"audioCodec,omitempty"` // Audio codec used in the video.
	VideoCodec string `json:"videoCodec,omitempty"` // Video codec used.
}

// AudioFacet provides metadata specific to items that are audio files.
type AudioFacet struct {
	Album             string `json:"album,omitempty"`             // Album name.
	Artist            string `json:"artist,omitempty"`            // Artist name.
	Bitrate           int    `json:"bitrate,omitempty"`           // Bitrate in bits per second.
	Duration          int64  `json:"duration,omitempty"`          // Duration in milliseconds.
	Genre             string `json:"genre,omitempty"`             // Genre.
	Title             string `json:"title,omitempty"`             // Title of the audio track.
	Track             int    `json:"track,omitempty"`             // Track number.
	Year              int    `json:"year,omitempty"`              // Year of recording.
	IsVariableBitrate bool   `json:"isVariableBitrate,omitempty"` // Whether the audio is variable bitrate.
}

// PhotoFacet provides EXIF metadata specific to items that are photos.
type PhotoFacet struct {
	TakenDateTime       time.Time `json:"takenDateTime,omitempty"`       // Timestamp when the photo was taken.
	CameraMake          string    `json:"cameraMake,omitempty"`          // Make of the camera used.
	CameraModel         string    `json:"cameraModel,omitempty"`         // Model of the camera used.
	FNumber             float64   `json:"fNumber,omitempty"`             // F-number (aperture).
	ExposureNumerator   int       `json:"exposureNumerator,omitempty"`   // Exposure time numerator.
	ExposureDenominator int       `json:"exposureDenominator,omitempty"` // Exposure time denominator.
	FocalLength         float64   `json:"focalLength,omitempty"`         // Focal length.
	Iso                 int       `json:"iso,omitempty"`                 // ISO speed.
}

// PackageFacet indicates that a DriveItem is a package, an alternate data stream used by some applications.
type PackageFacet struct {
	Type string `json:"type"` // Type of package (e.g., "oneNote").
}

// RemoteItemFacet indicates that a DriveItem is a link to an item on another drive.
type RemoteItemFacet struct {
	ID             string    `json:"id"`     // ID of the remote item.
	Name           string    `json:"name"`   // Name of the remote item.
	Size           int64     `json:"size"`   // Size of the remote item.
	WebURL         string    `json:"webUrl"` // URL to access the remote item.
	FileSystemInfo *struct { // File system info of the remote item.
		CreatedDateTime      time.Time `json:"createdDateTime"`
		LastModifiedDateTime time.Time `json:"lastModifiedDateTime"`
	} `json:"fileSystemInfo,omitempty"`
	Folder *FolderFacet `json:"folder,omitempty"` // If the remote item is a folder.
	File   *FileFacet   `json:"file,omitempty"`   // If the remote item is a file.
}

// DeletedFacet provides information about a deleted DriveItem.
type DeletedFacet struct {
	State string `json:"state"` // State of the deletion (e.g., "itemDeleted").
}

// UploadSession represents the response from creating an upload session for large files.
// It contains the URL to which chunks should be uploaded and session expiry information.
type UploadSession struct {
	UploadURL          string   `json:"uploadUrl"`                    // The URL to upload file chunks to.
	ExpirationDateTime string   `json:"expirationDateTime"`           // Timestamp when the upload session expires (ISO 8601 format).
	NextExpectedRanges []string `json:"nextExpectedRanges,omitempty"` // Byte ranges the server expects next (for resuming).
}

// DriveList represents a collection of Drive resources.
type DriveList struct {
	Value    []Drive `json:"value"`                     // The array of Drive resources.
	NextLink string  `json:"@odata.nextLink,omitempty"` // URL to the next page of results, if any.
}

// Drive represents a OneDrive drive resource. A drive can be a user's personal OneDrive,
// a OneDrive for Business drive, or a SharePoint document library.
type Drive struct {
	ID        string   `json:"id"`        // The unique identifier of the Drive.
	Name      string   `json:"name"`      // The name of the Drive.
	DriveType string   `json:"driveType"` // Type of the drive (e.g., "personal", "business", "documentLibrary").
	Owner     struct { // Information about the owner of the Drive.
		User *Identity `json:"user,omitempty"`
	} `json:"owner"`
	Quota struct { // Storage quota information for the Drive.
		Total     int64  `json:"total"`     // Total allowed storage in bytes.
		Used      int64  `json:"used"`      // Used storage in bytes.
		Remaining int64  `json:"remaining"` // Remaining storage in bytes.
		State     string `json:"state"`     // Quota state (e.g., "normal", "nearing", "critical", "exceeded").
	} `json:"quota"`
}

// DeviceCodeResponse holds the response from the OAuth2 device code endpoint.
// It provides the user with a code and a URL to complete authentication in a browser.
type DeviceCodeResponse struct {
	UserCode        string `json:"user_code"`        // The code the user needs to enter at the verification URI.
	DeviceCode      string `json:"device_code"`      // The code the application uses to poll for the token.
	VerificationURI string `json:"verification_uri"` // The URL the user should visit to enter the user_code.
	ExpiresIn       int    `json:"expires_in"`       // How long the device_code and user_code are valid, in seconds.
	Interval        int    `json:"interval"`         // Recommended polling interval in seconds.
	Message         string `json:"message"`          // User-friendly message with instructions.
}

// User represents a Microsoft Graph user resource.
// It contains basic profile information about a user.
type User struct {
	DisplayName       string `json:"displayName"`       // The user's display name.
	UserPrincipalName string `json:"userPrincipalName"` // The user's principal name (often their email address).
	ID                string `json:"id"`                // The unique identifier of the User.
}

// CopyOperationStatus represents the status of an asynchronous copy operation for a DriveItem.
// This is polled using the monitor URL returned when a copy is initiated.
type CopyOperationStatus struct {
	Status             string    `json:"status"`             // Current status: "notStarted", "inProgress", "completed", "failed", "waiting".
	PercentageComplete int       `json:"percentageComplete"` // Estimated percentage of completion (0-100).
	StatusDescription  string    `json:"statusDescription"`  // Human-readable description of the status.
	Error              *struct { // Details if the operation failed.
		Code    string `json:"code"`    // Error code (e.g., "itemNotFound").
		Message string `json:"message"` // Error message.
	} `json:"error,omitempty"`
	ResourceID       string `json:"resourceId,omitempty"`       // ID of the new item created by the copy operation.
	ResourceLocation string `json:"resourceLocation,omitempty"` // Deprecated: URL of the completed item. Use resourceId.
}

// CreateLinkRequest represents the request body for creating a sharing link for a DriveItem.
type CreateLinkRequest struct {
	Type     string `json:"type"`               // Type of link: "view" (read-only), "edit" (read-write), or "embed" (for web pages).
	Scope    string `json:"scope"`              // Scope of the link: "anonymous" (anyone with the link) or "organization" (members of the user's org).
	Password string `json:"password,omitempty"` // Optional password to protect the link.
	// ExpirationDateTime string `json:"expirationDateTime,omitempty"` // Optional: ISO 8601 format for link expiry.
}

// SharingLink represents a sharing link created for a DriveItem.
// It contains the URL and properties of the link.
type SharingLink struct {
	ID          string   `json:"id"`                    // Unique ID of the permission resource representing this link.
	Roles       []string `json:"roles,omitempty"`       // Roles associated with the link (e.g., "read", "write").
	ShareId     string   `json:"shareId,omitempty"`     // Unique ID of the share action.
	HasPassword bool     `json:"hasPassword,omitempty"` // True if the link is password protected.
	Link        struct { // Details about the link itself.
		Type        string    `json:"type"`              // "view", "edit", or "embed".
		Scope       string    `json:"scope"`             // "anonymous" or "organization".
		WebUrl      string    `json:"webUrl"`            // The actual sharing URL.
		WebHtml     string    `json:"webHtml,omitempty"` // For "embed" links, the HTML snippet.
		Application *struct { // Application that created the link, if applicable.
			Id          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"application,omitempty"`
	} `json:"link"`
	ExpirationDateTime string `json:"expirationDateTime,omitempty"` // ISO 8601 timestamp when the link expires.
}

// DeltaResponse represents the response from a delta query on a drive.
// It contains a list of changed items and links for pagination or continuing the delta sync.
type DeltaResponse struct {
	Value     []DriveItem `json:"value"`                      // Items that have changed since the last delta token.
	DeltaLink string      `json:"@odata.deltaLink,omitempty"` // URL to get changes since this response (contains the next delta token).
	NextLink  string      `json:"@odata.nextLink,omitempty"`  // URL to get the next page of current changes, if the result is paged.
}

// DriveItemVersion represents a specific version of a file in OneDrive.
type DriveItemVersion struct {
	ID                   string    `json:"id"`                   // The unique identifier of the version.
	LastModifiedDateTime time.Time `json:"lastModifiedDateTime"` // Timestamp when this version was created.
	Size                 int64     `json:"size"`                 // Size of this version in bytes.
	LastModifiedBy       struct {  // User or application that created this version.
		User *Identity `json:"user,omitempty"`
	} `json:"lastModifiedBy"`
	// Content              io.ReadCloser `json:"content,omitempty"` // Not typically part of metadata; content is fetched separately.
}

// DriveItemVersionList represents a collection of DriveItemVersion resources.
type DriveItemVersionList struct {
	Value    []DriveItemVersion `json:"value"`                     // The array of file versions.
	NextLink string             `json:"@odata.nextLink,omitempty"` // URL to the next page of versions, if any.
}

// Paging represents pagination options for API calls that return lists of items.
// It allows controlling the number of items per page and fetching subsequent pages.
type Paging struct {
	// Top specifies the maximum number of items to return in a single page.
	// If 0, the Microsoft Graph API default is used (often 200 items).
	Top int `json:"top,omitempty"`
	// FetchAll, if true, instructs the SDK's helper functions (like collectAllPages)
	// to ignore `Top` and `NextLink` and automatically follow all `@odata.nextLink` URLs
	// to retrieve all items across all pages. Use with caution for very large result sets.
	FetchAll bool `json:"fetchAll,omitempty"`
	// NextLink, if provided, specifies the exact `@odata.nextLink` URL to start fetching from.
	// This is used to resume pagination or get a specific subsequent page.
	NextLink string `json:"nextLink,omitempty"`
}

// Activity represents an action or event that occurred on a DriveItem or within a container (like a drive).
// It provides details about the action, the actor, and the time of occurrence.
type Activity struct {
	ID     string   `json:"id"` // Unique ID of the activity.
	Action struct { // Describes the type of action. Only one of these fields will typically be non-nil.
		Comment *struct{} `json:"comment,omitempty"` // A comment was added.
		Create  *struct{} `json:"create,omitempty"`  // An item was created.
		Delete  *struct{} `json:"delete,omitempty"`  // An item was deleted.
		Edit    *struct { // An item was edited.
			NewVersion string `json:"newVersion,omitempty"` // ID of the new version created by the edit.
		} `json:"edit,omitempty"`
		Mention *struct { // A user was mentioned.
			Mentionees []struct {
				User *Identity `json:"user,omitempty"`
			} `json:"mentionees,omitempty"`
		} `json:"mention,omitempty"`
		Move   *struct{} `json:"move,omitempty"` // An item was moved.
		Rename *struct { // An item was renamed.
			OldName string `json:"oldName,omitempty"` // The previous name of the item.
		} `json:"rename,omitempty"`
		Restore *struct{} `json:"restore,omitempty"` // An item was restored from the recycle bin.
		Share   *struct{} `json:"share,omitempty"`   // An item was shared.
		Version *struct { // A new version of an item was created (e.g., explicit versioning action).
			NewVersion string `json:"newVersion,omitempty"`
		} `json:"version,omitempty"`
	} `json:"action"`
	Actor struct { // The user or application that performed the action.
		User *Identity `json:"user,omitempty"`
	} `json:"actor"`
	Times struct { // Timestamps related to the activity.
		RecordedTime time.Time `json:"recordedTime"` // When the activity was recorded by the service.
	} `json:"times"`
	DriveItem *DriveItem `json:"driveItem,omitempty"` // The DriveItem this activity pertains to, if applicable.
}

// ActivityList represents a collection of Activity resources, typically with pagination information.
type ActivityList struct {
	Value    []Activity `json:"value"`                     // The array of activities.
	NextLink string     `json:"@odata.nextLink,omitempty"` // URL to the next page of activities, if any.
}

// Thumbnail represents a single thumbnail image for a DriveItem, with size and URL.
type Thumbnail struct {
	Height int    `json:"height"` // Height of the thumbnail in pixels.
	Width  int    `json:"width"`  // Width of the thumbnail in pixels.
	URL    string `json:"url"`    // URL to download the thumbnail image.
}

// ThumbnailSet represents a collection of thumbnails of different predefined sizes for a DriveItem.
// Common sizes include "small", "medium", and "large".
type ThumbnailSet struct {
	ID     string     `json:"id"`               // ID of the thumbnail set (usually "0").
	Small  *Thumbnail `json:"small,omitempty"`  // Small-sized thumbnail.
	Medium *Thumbnail `json:"medium,omitempty"` // Medium-sized thumbnail.
	Large  *Thumbnail `json:"large,omitempty"`  // Large-sized thumbnail.
	Source *Thumbnail `json:"source,omitempty"` // Thumbnail representing the original item, if applicable.
}

// ThumbnailSetList represents a collection of ThumbnailSet resources.
// Typically, a DriveItem has one ThumbnailSet (ID "0") containing various sizes.
type ThumbnailSetList struct {
	Value    []ThumbnailSet `json:"value"`                     // The array of thumbnail sets.
	NextLink string         `json:"@odata.nextLink,omitempty"` // URL to the next page, if applicable (rare for thumbnails).
}

// PreviewRequest represents the request body for generating a preview of a DriveItem.
// Optional parameters can specify the page number or zoom level for the preview.
type PreviewRequest struct {
	Page string  `json:"page,omitempty"` // Page number (e.g., "1") or name to preview.
	Zoom float64 `json:"zoom,omitempty"` // Zoom level (e.g., 1.0 for 100%, 0.5 for 50%).
}

// PreviewResponse represents the response from the item preview API.
// It provides URLs that can be used to embed or display a preview of the item.
type PreviewResponse struct {
	GetURL         string `json:"getUrl,omitempty"`         // URL to GET the preview content directly.
	PostURL        string `json:"postUrl,omitempty"`        // URL to POST to for preview (e.g., with form parameters).
	PostParameters string `json:"postParameters,omitempty"` // Parameters to include in the POST request if using PostURL.
}

// Permission represents a sharing permission granted on a DriveItem.
// It details who has access, what type of access, and how it was granted (e.g., direct or via a link).
type Permission struct {
	ID                 string   `json:"id"`                           // Unique ID of the permission.
	Roles              []string `json:"roles"`                        // Roles granted (e.g., "read", "write", "owner").
	ShareID            string   `json:"shareId,omitempty"`            // ID of the share action, if this permission is for a sharing link.
	ExpirationDateTime string   `json:"expirationDateTime,omitempty"` // ISO 8601 timestamp when the permission expires.
	HasPassword        bool     `json:"hasPassword,omitempty"`        // True if the permission (usually a link) is password-protected.
	// PermissionScope string   `json:"scope,omitempty"` // Deprecated: Use Link.Scope or GrantedToV2 for user scope.
	GrantedToV2 *struct { // Information about the user or group granted this permission directly.
		User     *Identity `json:"user,omitempty"`
		SiteUser *Identity `json:"siteUser,omitempty"` // For SharePoint site users.
		// Group    *Identity `json:"group,omitempty"` // If permission is granted to a group.
		// Application *Identity `json:"application,omitempty"` // If granted to an application.
	} `json:"grantedToV2,omitempty"`
	GrantedToIdentitiesV2 []struct { // Collection of identities this permission is granted to.
		User     *Identity `json:"user,omitempty"`
		SiteUser *Identity `json:"siteUser,omitempty"`
		// Group    *Identity `json:"group,omitempty"`
		// Application *Identity `json:"application,omitempty"`
	} `json:"grantedToIdentitiesV2,omitempty"`
	Link *struct { // If this permission is for a sharing link, this contains link details.
		Type        string    `json:"type"`              // "view", "edit", "embed".
		Scope       string    `json:"scope"`             // "anonymous", "organization", "users" (for specific users link).
		WebURL      string    `json:"webUrl"`            // The sharing URL.
		WebHTML     string    `json:"webHtml,omitempty"` // HTML for embedding (for "embed" type).
		Application *struct { // Application that created the link.
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"application,omitempty"`
		PreventsDownload bool `json:"preventsDownload,omitempty"` // True if the link prevents downloading the item.
	} `json:"link,omitempty"`
	InheritedFrom *struct { // If this permission is inherited from a parent item.
		DriveID string `json:"driveId"` // Drive ID of the item from which permission is inherited.
		ID      string `json:"id"`      // Item ID from which permission is inherited.
		Path    string `json:"path"`    // Path of the item from which permission is inherited.
	} `json:"inheritedFrom,omitempty"`
	Invitation *struct { // If this permission was granted via an invitation.
		SignInRequired bool   `json:"signInRequired"`  // True if the invited user must sign in.
		Email          string `json:"email,omitempty"` // Email address of the invited user.
	} `json:"invitation,omitempty"`
}

// PermissionList represents a collection of Permission resources for a DriveItem.
type PermissionList struct {
	Value    []Permission `json:"value"`                     // The array of permissions.
	NextLink string       `json:"@odata.nextLink,omitempty"` // URL to the next page of permissions, if any.
}

// InviteRequest represents the request body for inviting users to access a DriveItem.
type InviteRequest struct {
	Recipients []struct { // List of recipients to invite.
		Email    string `json:"email,omitempty"`    // Email address of the recipient.
		ObjectID string `json:"objectId,omitempty"` // Azure AD object ID of the recipient (user or group).
	} `json:"recipients"`
	Message              string   `json:"message,omitempty"`              // Optional custom message for the invitation email.
	RequireSignIn        bool     `json:"requireSignIn,omitempty"`        // If true, recipient must sign in to access. Defaults to true.
	SendInvitation       bool     `json:"sendInvitation,omitempty"`       // If true, an email invitation is sent. Defaults to true.
	Roles                []string `json:"roles"`                          // Roles to grant (e.g., "read", "write").
	ExpirationDateTime   string   `json:"expirationDateTime,omitempty"`   // Optional: ISO 8601 timestamp for when the permission expires.
	Password             string   `json:"password,omitempty"`             // Optional: Password for the sharing link created by the invitation.
	RetainInheritedRoles bool     `json:"retainInheritedRoles,omitempty"` // If true, existing inherited roles are kept; otherwise, they might be replaced.
}

// InviteResponse represents the response from an invite operation.
// It typically returns a list of Permission resources created for the invited users.
type InviteResponse struct {
	Value []Permission `json:"value"` // Permissions created for the invited users/groups.
}

// UpdatePermissionRequest represents the request body for updating an existing permission on a DriveItem.
type UpdatePermissionRequest struct {
	Roles              []string `json:"roles,omitempty"`              // New roles to set (e.g., "read", "write", "owner").
	ExpirationDateTime string   `json:"expirationDateTime,omitempty"` // New expiration timestamp (ISO 8601 format).
	Password           string   `json:"password,omitempty"`           // New password for a link-based permission.
}
