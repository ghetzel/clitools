package main

type ytdlInfo struct {
	ID              string `json:"id"`
	UploadDatestamp string `json:"upload_date"`
	Title           string `json:"title"`
	FullTitle       string `json:"fulltitle"`
	Description     string `json:"description"`
}

func (self ytdlInfo) Field(name string) interface{} {
	switch name {
	case `year`:
		if len(self.UploadDatestamp) >= 4 {
			return self.UploadDatestamp[0:4]
		}
	case `month`:
		if len(self.UploadDatestamp) >= 6 {
			return self.UploadDatestamp[4:6]
		}
	case `day`:
		if len(self.UploadDatestamp) >= 8 {
			return self.UploadDatestamp[6:8]
		}
	case `id`:
		return self.ID
	case `title`:
		switch v := self.Title; v {
		case ``, `_`:
			return self.FullTitle
		default:
			return self.Title
		}
	case `description`:
		return self.Description
	}

	return nil
}
