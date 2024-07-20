package main

type FileInfo struct {
	FileName  string
	FilePath  string
	IsDisable bool
	Hash      string
}

type RJson struct {
	File []FileReInfo `json:"file"`
}

type FileReInfo struct {
	FileName string `json:"filename"`
	Hash     string `json:"hash"`
}

type Process struct {
	FilePath    string
	DownloadUrl string
	Status      int
}

type PathArray struct {
	Folder string `json:"folder"`
}

type ListResult struct {
	Ver    int      `json:"ver"`
	Folder []string `json:"folder"`
}
