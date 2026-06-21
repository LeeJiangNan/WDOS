package model

// CRIPCallback CRIP Callback 推送的完整 JSON 结构
type CRIPCallback struct {
	SnowflakeID    string              `json:"snowflake_id" binding:"required"`
	AnalysisJobID  string              `json:"analysis_job_id"`
	Timestamp      string              `json:"timestamp"`
	CameraID       int                 `json:"camera_id"`
	CameraUUID     string              `json:"camera_uuid"`
	CameraName     string              `json:"camera_name"`
	CameraGroup    []string            `json:"camera_group"`
	CameraTypes    []int               `json:"camera_types"`
	ChannelID      string              `json:"channel_id"`
	GPS            string              `json:"gps"`
	StreamURL      string              `json:"stream_url"`
	OnlineStatus   int                 `json:"online_status"`
	AlgorithmID    int                 `json:"algorithm_id"`
	AlgorithmName  string              `json:"algorithm_name"`
	AlgorithmNameEn string             `json:"algorithm_name_en"`
	Degree         string              `json:"degree"`
	AlarmPicURL    string              `json:"alarm_pic_url"`
	AlarmPicData   string              `json:"alarm_pic_data"`
	AlarmPicName   string              `json:"alarm_pic_name"`
	SrcPicURL      string              `json:"src_pic_url"`
	SrcPicData     string              `json:"src_pic_data"`
	SrcPicName     string              `json:"src_pic_name"`
	VideoURL       string              `json:"video_url"`
	VideoName      string              `json:"video_name"`
	ImageWidth     int                 `json:"image_width"`
	ImageHeight    int                 `json:"image_height"`
	Extra          string              `json:"extra"`
	Members        []CRIPMember        `json:"members"`
	ResultData     []CRIPResultData    `json:"result_data"`
}

// CRIPMember 人员信息（人脸识别场景）
type CRIPMember struct {
	UserID   string  `json:"user_id"`
	UserName string  `json:"user_name"`
	Tag      string  `json:"tag"`
	Score    float64 `json:"score"`
	Photo    string  `json:"photo"`
	Role     string  `json:"role"`
}

// CRIPResultData 报警详细数据
type CRIPResultData struct {
	AlgorithmName   string              `json:"algorithm_name"`
	AlgorithmEnName string              `json:"algorithm_en_name"`
	Degree          string              `json:"degree"`
	TaskID          int                 `json:"task_id"`
	ResultData      CRIPTaskResultData  `json:"result_data"`
}

// CRIPTaskResultData 分析结果数据
type CRIPTaskResultData struct {
	TaskID     int               `json:"task_id"`
	TaskResult CRIPTaskResult    `json:"task_result"`
}

// CRIPTaskResult 任务结果
type CRIPTaskResult struct {
	ClassID    int                `json:"class_id"`
	ExtraData  string             `json:"extra_data"`
	ObjectList []CRIPDetectObject `json:"object_list"`
	Score      float64            `json:"score"`
}

// CRIPDetectObject 检测目标
type CRIPDetectObject struct {
	ClassID   int     `json:"class_id"`
	Rect      CRIPRect `json:"rect"`
	Score     float64 `json:"score"`
	ExtraData string  `json:"extra_data"`
}

// CRIPRect 检测框坐标
type CRIPRect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// CallbackResponse Callback 接口返回
type CallbackResponse struct {
	Action      string `json:"action"`       // created / suppressed / locked / ignored
	WorkOrderID uint64 `json:"work_order_id"`
	Suppressed   bool   `json:"suppressed"`
	Reason      string `json:"reason"`
}
