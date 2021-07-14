package forms

import "cloudiac/portal/models"

type CreateTaskCommentForm struct {
	PageForm

	TaskId  models.Id `json:"taskId" form:"taskId" binding:"required"`
	Comment string    `json:"comment" form:"comment" binding:"required"`
}

type SearchTaskCommentForm struct {
	PageForm
	TaskId models.Id `json:"taskId" form:"taskId" `
}