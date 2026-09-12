package api

type Audio struct {
	ID                int    `json:"id"`
	OwnerID           int    `json:"owner_id"`
	Artist            string `json:"artist"`
	Title             string `json:"title"`
	URL               string `json:"url"`
	ContentRestricted int    `json:"content_restricted,omitempty"`
}

type Playlist struct {
	ID      int    `json:"id"`
	OwnerID int    `json:"owner_id"`
	Title   string `json:"title"`
}

type VkAudioResponse struct {
	Response *struct {
		Count int     `json:"count"`
		Items []Audio `json:"items"`
	} `json:"response"`
	Error *VkError `json:"error"`
}

type VkPlaylistsResponse struct {
	Response *struct {
		Count int        `json:"count"`
		Items []Playlist `json:"items"`
	} `json:"response"`
	Error *VkError `json:"error"`
}

type VkAudioByIdResponse struct {
	Response []Audio  `json:"response"`
	Error    *VkError `json:"error"`
}

type VkError struct {
	ErrorCode int    `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}
