package forms

type CreateTaskForm struct {
	BaseForm

	TaskName      string `json:"taskName" form:"taskName"`
	CtServiceIp   string `json:"ctServiceIp" form:"ctServiceIp" binding:"required"`
	CtServicePort string `json:"ctServicePort" form:"ctServicePort" binding:"required"`
	CtServiceId   string `json:"ctServiceId" form:"ctServiceId" binding:"required"`
	TemplateId    uint   `json:"templateId" form:"templateId" binding:"required"`
	TemplateGuid  string `json:"templateGuid" form:"templateGuid" binding:"required"`
	TaskType      string `json:"taskType" form:"taskType" binding:"required"`
}

type DetailTaskForm struct {
	BaseForm
	TaskId uint `json:"taskId" form:"taskId" binding:"required"`
}

type SearchTaskForm struct {
	BaseForm
	TemplateId uint   `json:"templateId" form:"templateId" binding:"required"`
	Q          string `form:"q" json:"q" binding:""`
	Status     string `form:"status" json:"status"`
}