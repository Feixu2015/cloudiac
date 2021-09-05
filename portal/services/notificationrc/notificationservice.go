package notificationrc

import (
	"cloudiac/configs"
	"cloudiac/portal/consts"
	"cloudiac/portal/libs/db"
	"cloudiac/portal/models"
	"cloudiac/utils"
	"cloudiac/utils/logs"
	"cloudiac/utils/mail"
	"fmt"
	"time"
)

type NotificationService struct {
	Tpl       *models.Template     `json:"tpl" form:"tpl" `
	Project   *models.Project      `json:"project" form:"project" `
	Org       *models.Organization `json:"org" form:"org" `
	OrgId     models.Id            `json:"orgId" form:"orgId" `
	ProjectId models.Id            `json:"projectId" form:"projectId" `
	Env       *models.Env          `json:"env" form:"env" `
	Task      *models.Task         `json:"task" form:"task" `
	EventType string               `json:"eventType" form:"eventType" `
}

type NotificationOptions struct {
	Tpl       *models.Template     `json:"tpl" form:"tpl" `
	Project   *models.Project      `json:"project" form:"project" `
	Org       *models.Organization `json:"org" form:"org" `
	OrgId     models.Id            `json:"orgId" form:"orgId" `
	ProjectId models.Id            `json:"projectId" form:"projectId" `
	Env       *models.Env          `json:"env" form:"env" `
	Task      *models.Task         `json:"task" form:"task" `
	EventType string               `json:"eventType" form:"eventType" `
}

func NewNotificationService(options *NotificationOptions) NotificationService {
	return NotificationService{
		OrgId:     options.OrgId,
		ProjectId: options.ProjectId,
		Env:       options.Env,
		Task:      options.Task,
		Tpl:       options.Tpl,
		Project:   options.Project,
		Org:       options.Org,
		EventType: options.EventType,
	}
}

func (ns *NotificationService) SendMessage() {
	notifications, tplNotificationTemplate, markdownNotificationTemplate := ns.FindNotificationsAndMessageTpl()
	if len(notifications) == 0 {
		return
	}
	u := models.User{}
	_ = db.Get().Where("id = ?", ns.Task.CreatorId).First(&u)

	data := struct {
		Creator      string
		OrgName      string
		ProjectName  string
		TemplateName string
		Revision     string
		EnvName      string
		Addr         string
		ResAdded     *int
		ResChanged   *int
		ResDestroyed *int
		Message      string
	}{
		Creator:      u.Name,
		OrgName:      ns.Org.Name,
		ProjectName:  ns.Project.Name,
		TemplateName: ns.Tpl.Name,
		Revision:     ns.Tpl.RepoRevision,
		EnvName:      ns.Env.Name,
		//http://{{addr}}/org/{{orgId}}/project/{{ProjectId}}/m-project-env/detail/{{envId}}/deployHistory/task/{{TaskId}}
		Addr:         fmt.Sprintf("%s/org/%s/project/%s/m-project-env/detail/%s/deployHistory/task/%s", configs.Get().Portal.Address, ns.Org.Id, ns.ProjectId, ns.Env.Id, ns.Task.Id),
		ResAdded:     ns.Task.Result.ResAdded,
		ResChanged:   ns.Task.Result.ResChanged,
		ResDestroyed: ns.Task.Result.ResDestroyed,
		Message:      ns.Task.Message,
	}

	// 获取消息通知模板
	markdownNotificationTemplate = utils.SprintTemplate(markdownNotificationTemplate, data)
	tplNotificationTemplate = utils.SprintTemplate(tplNotificationTemplate, data)
	userIds := make([]string, 0)
	// 判断消息类型，下发至的消息通道
	for _, notification := range notifications {
		if notification.Type == models.NotificationTypeEmail {
			userIds = append(userIds, notification.UserIds...)
			continue
		}
		go func(notification models.Notification) {
			switch notification.Type {
			case models.NotificationTypeDingTalk:
				ns.SendDingTalkMessage(notification, markdownNotificationTemplate)
			case models.NotificationTypeWebhook:
				ns.SendWebhookMessage(notification, markdownNotificationTemplate)
			case models.NotificationTypeWeChat:
				ns.SendWechatMessage(notification, markdownNotificationTemplate)
			case models.NotificationTypeSlack:
				ns.SendSlackMessage(notification, markdownNotificationTemplate)
			}
		}(notification)
		time.Sleep(time.Second)
	}
	userIds = utils.RemoveDuplicateElement(userIds)

	func(userIds []string) {
		// 获取用户邮箱列表
		users := make([]models.User, 0)
		_ = db.Get().Where("id in (?)", userIds).Find(users)
		for _, v := range users {
			// 单个用户发送邮件，避免暴露其他用户邮箱
			go ns.SendEmailMessage([]string{
				v.Email,
			}, tplNotificationTemplate)
			// 避免并发量太大
			time.Sleep(time.Millisecond)
		}
	}(userIds)
}

func (ns *NotificationService) SendDingTalkMessage(n models.Notification, message string) {
	dingTalk := NewDingTalkRobot(n.Url, n.Secret)
	if err := dingTalk.SendMarkdownMessage(consts.NotificationMessageTitle, message, nil, false); err != nil {
		logs.Get().Errorf("send dingtalk message err: %v", err)
	}
}

func (ns *NotificationService) SendWechatMessage(n models.Notification, message string) {
	wechat := WeChatRobot{Url: n.Url}
	if _, err := wechat.SendMarkdown(message); err != nil {
		logs.Get().Errorf("send wechat message err: %v", err)

	}
}

func (ns *NotificationService) SendWebhookMessage(n models.Notification, message string) {
	w := Webhook{Url: n.Url}
	if err := w.Send(message); err != nil {
		logs.Get().Errorf("send webhook message err: %v", err)
	}
}

func (ns *NotificationService) SendSlackMessage(n models.Notification, message string) {
	if errs := SendSlack(n.Url, Payload{Text: message, Markdown: true}); len(errs) != 0 {
		logs.Get().Errorf("send slack message err: %v", errs)
	}

}

func (ns *NotificationService) SendEmailMessage(emails []string, message string) {
	if len(emails) < 1 {
		return
	}
	if err := mail.SendMail(emails, consts.NotificationMessageTitle, message); err != nil {
		logs.Get().Errorf("send mail message err: %v", err)
	}
}

func (ns *NotificationService) FindNotificationsAndMessageTpl() ([]models.Notification, string, string) {
	orgNotification := make([]models.Notification, 0)
	projectNotification := make([]models.Notification, 0)
	notifications := make([]models.Notification, 0)
	dbSess := db.Get().Debug().Where("org_id = ?", ns.OrgId).
		Joins(fmt.Sprintf("left join %s as ne on %s.id = ne.notification_id",
			models.NotificationEvent{}.TableName(), models.Notification{}.TableName())).
		Where("ne.event_type = ?", ns.EventType)
	var (
		tplNotificationTemplate      string
		markdownNotificationTemplate string
	)

	switch ns.EventType {
	case models.EventTaskRunning:
		tplNotificationTemplate = consts.IacTaskRunning
		markdownNotificationTemplate = consts.IacTaskRunningMarkdown
	case models.EventTaskApproving:
		tplNotificationTemplate = consts.IacTaskApprovingTpl
		markdownNotificationTemplate = consts.IacTaskApprovingMarkdown
	case models.EventTaskFailed:
		tplNotificationTemplate = consts.IacTaskFailedTpl
		markdownNotificationTemplate = consts.IacTaskFailedMarkdown
	case models.EventTaskComplete:
		tplNotificationTemplate = consts.IacTaskCompleteTpl
		markdownNotificationTemplate = consts.IacTaskCompleteMarkdown
	}

	// 查询需要组织下需要通知的人
	if err := dbSess.
		Where("project_id = '' or project_id is null or project_id = ?", ns.ProjectId).
		Find(&orgNotification); err != nil {
		return notifications, tplNotificationTemplate, markdownNotificationTemplate
	}
	// 将需要通知的数据进行整理
	notifications = append(notifications, orgNotification...)
	notifications = append(notifications, projectNotification...)
	return notifications, tplNotificationTemplate, markdownNotificationTemplate
}