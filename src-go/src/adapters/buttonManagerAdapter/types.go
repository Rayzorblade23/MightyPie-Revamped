package buttonManagerAdapter


type WindowInfo_Message struct {
    Title    string `json:"Title"`
    ExeName  string `json:"ExeName"`
    ExePath  string `json:"ExePath"`
    AppName  string `json:"AppName"`
    Instance int    `json:"Instance"`
}

type WindowInfo struct {
    Title    string
    ExeName  string
    ExePath  string
    AppName  string
    Instance int   
}

type WindowsUpdate_Message map[int]WindowInfo_Message

type WindowMapping map[int]WindowInfo
