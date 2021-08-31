package apps

import (
	"cloudiac/common"
	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/libs/page"
	"cloudiac/portal/models"
	"cloudiac/portal/models/forms"
	"cloudiac/portal/services"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// CreatePolicy 创建策略
func CreatePolicy(c *ctx.ServiceContext, form *forms.CreatePolicyForm) (*models.Policy, e.Error) {
	c.AddLogField("action", fmt.Sprintf("create policy %s", form.Name))

	entryName, resourceType, policyType, err := parseRegoHeader(form.Rego)
	if err != nil {
		return nil, e.New(err.Code(), err, http.StatusBadRequest)
	}

	p := models.Policy{
		Name:          form.Name,
		CreatorId:     c.UserId,
		FixSuggestion: form.FixSuggestion,
		Severity:      form.Severity,
		Rego:          form.Rego,
		Entry:         entryName,
		ResourceType:  resourceType,
		PolicyType:    policyType,
	}
	refId, err := services.GetPolicyReferenceId(c.DB(), &p)
	if err != nil {
		return nil, e.New(e.DBError, err, http.StatusInternalServerError)
	}
	p.ReferenceId = refId

	policy, err := services.CreatePolicy(c.DB(), &p)
	if err != nil && err.Code() == e.PolicyAlreadyExist {
		return nil, e.New(err.Code(), err, http.StatusBadRequest)
	} else if err != nil {
		c.Logger().Errorf("error creating policy, err %s", err)
		return nil, e.AutoNew(err, e.DBError)
	}

	return policy, nil
}

//parseRegoHeader 解析 rego 脚本获取入口，云商类型和资源类型
func parseRegoHeader(rego string) (entry string, policyType string, resType string, err e.Error) {
	regex := regexp.MustCompile("(?m)^##entry (.*)$")
	match := regex.FindStringSubmatch(rego)
	if len(match) == 2 {
		entry = strings.TrimSpace(match[1])
	} else {
		return "", "", "", e.New(e.PolicyRegoMissingComment)
	}

	regex = regexp.MustCompile("(?m)^##policyType (.*)$")
	match = regex.FindStringSubmatch(rego)
	if len(match) == 2 {
		policyType = strings.TrimSpace(match[1])
	} else {
		return "", "", "", e.New(e.PolicyRegoMissingComment)
	}

	regex = regexp.MustCompile("(?m)^##resourceType (.*)$")
	match = regex.FindStringSubmatch(rego)
	if len(match) == 2 {
		resType = strings.TrimSpace(match[1])
	} else {
		return "", "", "", e.New(e.PolicyRegoMissingComment)
	}
	return
}

// ScanTemplate 扫描云模板策略
func ScanTemplate(c *ctx.ServiceContext, form *forms.ScanTemplateForm) (*models.ScanTask, e.Error) {
	c.AddLogField("action", fmt.Sprintf("create scan template %s", form.Id))
	if c.OrgId == "" {
		return nil, e.New(e.BadRequest, http.StatusBadRequest)
	}
	// TODO: 检查云模板访问权限

	tx := c.Tx()
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	// 模板检查
	tplQuery := services.QueryWithOrgId(tx, c.OrgId)
	tpl, err := services.GetTemplateById(tplQuery, form.Id)
	if err != nil && err.Code() == e.TemplateNotExists {
		return nil, e.New(err.Code(), err, http.StatusBadRequest)
	} else if err != nil {
		c.Logger().Errorf("error get template, err %s", err)
		return nil, e.New(e.DBError, err, http.StatusInternalServerError)
	}

	// 创建任务
	runnerId, err := services.GetDefaultRunnerId()
	if err != nil {
		return nil, e.New(err.Code(), err, http.StatusInternalServerError)
	}
	task, err := services.CreateScanTask(tx, tpl, nil, models.ScanTask{
		Name:      models.ScanTask{}.GetTaskNameByType(models.TaskTypeScan),
		CreatorId: c.UserId,
		Revision:  tpl.RepoRevision,
		BaseTask: models.BaseTask{
			Type:        models.TaskTypeScan,
			Flow:        models.TaskFlow{},
			StepTimeout: common.TaskStepTimeoutDuration,
			RunnerId:    runnerId,
		},
	})
	if err != nil {
		_ = tx.Rollback()
		c.Logger().Errorf("error creating scan task, err %s", err)
		return nil, e.New(err.Code(), err, http.StatusInternalServerError)
	}

	if _, err := tx.Save(task); err != nil {
		_ = tx.Rollback()
		c.Logger().Errorf("error save env, err %s", err)
		return nil, e.New(e.DBError, err, http.StatusInternalServerError)
	}

	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		c.Logger().Errorf("error commit env, err %s", err)
		return nil, e.New(e.DBError, err)
	}

	return task, nil
}

type PolicyResp struct {
	models.Policy
	GroupName string `json:"groupName" form:"groupName" `
}

// SearchPolicy 查询策略组列表
func SearchPolicy(c *ctx.ServiceContext, form *forms.SearchPolicyForm) (interface{}, e.Error) {
	query := services.SearchPolicy(c.DB(), form)
	pg := make([]PolicyResp, 0)
	p := page.New(form.CurrentPage(), form.PageSize(), query)
	if err := p.Scan(&pg); err != nil {
		return nil, e.New(e.DBError, err)
	}

	return pg, nil
}

// UpdatePolicy 修改策略组
func UpdatePolicy(c *ctx.ServiceContext, form *forms.UpdatePolicyForm) (interface{}, e.Error) {
	attr := models.Attrs{}
	if form.HasKey("name") {
		attr["name"] = form.Name
	}

	if form.HasKey("fixSuggestion") {
		attr["fixSuggestion"] = form.FixSuggestion
	}

	if form.HasKey("severity") {
		attr["severity"] = form.Severity
	}

	if form.HasKey("rego") {
		attr["rego"] = form.Rego
	}

	if form.HasKey("status") {
		attr["status"] = form.Status
	}

	pg := models.Policy{}
	pg.Id = form.Id
	if _, err := services.UpdatePolicy(c.DB(), &pg, attr); err != nil {
		return nil, err
	}
	return nil, nil
}

// DeletePolicy 删除策略组
func DeletePolicy(c *ctx.ServiceContext, form *forms.DeletePolicyForm) (interface{}, e.Error) {
	return services.DeletePolicy(c.DB(), form.Id)

}

// DetailPolicy 查询策略组详情
func DetailPolicy(c *ctx.ServiceContext, form *forms.DetailPolicyForm) (interface{}, e.Error) {
	return services.DetailPolicy(c.DB(), form.Id)
}

// CreatePolicySuppress 策略屏蔽
func CreatePolicySuppress(c *ctx.ServiceContext, form *forms.CreatePolicyShieldForm) (interface{}, e.Error) {
	return services.CreatePolicySuppress()
}

func SearchPolicySuppress(c *ctx.ServiceContext, form *forms.SearchPolicySuppressForm) (interface{}, e.Error) {
	return services.SearchPolicySuppress()
}

func DeletePolicySuppress(c *ctx.ServiceContext, form *forms.DeletePolicySuppressForm) (interface{}, e.Error) {
	return services.DeletePolicySuppress()
}

type RespPolicyTpl struct {
	TplName         string    `json:"tplName" form:"tplName" `
	TplId           models.Id `json:"tplId" form:"tplId" `
	RepoAddr        string    `json:"repoAddr" form:"repoAddr" `
	PolicyGroupName string    `json:"policyGroupName" form:"policyGroupName" `
	PolicyGroupId   models.Id `json:"policyGroupId" form:"policyGroupId" `
	Enabled         bool      `json:"enabled" example:"true"`          //是否启用
	GroupStatus     string    `json:"groupStatus" form:"groupStatus" ` //状态 todo 不确认字段
}

func SearchPolicyTpl(c *ctx.ServiceContext, form *forms.SearchPolicyTplForm) (interface{}, e.Error) {
	return services.SearchPolicyTpl()
}

func UpdatePolicyTpl(c *ctx.ServiceContext, form *forms.UpdatePolicyTplForm) (interface{}, e.Error) {
	return services.UpdatePolicyTpl()
}

func DetailPolicyTpl(c *ctx.ServiceContext, form *forms.DetailPolicyTplForm) (interface{}, e.Error) {
	return services.DetailPolicyTpl()
}

type RespPolicyEnv struct {
	TplName         string    `json:"tplName" form:"tplName" `
	TplId           models.Id `json:"tplId" form:"tplId" `
	EnvName         string    `json:"envName" form:"envName" `
	EnvId           models.Id `json:"envId" form:"envId" `
	RepoAddr        string    `json:"repoAddr" form:"repoAddr" `
	PolicyGroupName string    `json:"policyGroupName" form:"policyGroupName" `
	PolicyGroupId   models.Id `json:"policyGroupId" form:"policyGroupId" `
	Enabled         bool      `json:"enabled" example:"true"`          //是否启用
	GroupStatus     string    `json:"groupStatus" form:"groupStatus" ` //状态 todo 不确认字段
}

func SearchPolicyEnv(c *ctx.ServiceContext, form *forms.SearchPolicyEnvForm) (interface{}, e.Error) {
	return services.SearchPolicyEnv()
}

func UpdatePolicyEnv(c *ctx.ServiceContext, form *forms.UpdatePolicyEnvForm) (interface{}, e.Error) {
	return services.UpdatePolicyEnv()
}

func DetailPolicyEnv(c *ctx.ServiceContext, form *forms.DetailPolicyEnvForm) (interface{}, e.Error) {
	return services.DetailPolicyEnv()
}

func PolicyError(c *ctx.ServiceContext, form *forms.PolicyErrorForm) (interface{}, e.Error) {
	return services.PolicyError()
}

func PolicyReference(c *ctx.ServiceContext, form *forms.PolicyReferenceForm) (interface{}, e.Error) {
	return services.PolicyReference()
}

func PolicyRepo(c *ctx.ServiceContext, form *forms.PolicyRepoForm) (interface{}, e.Error) {
	return services.PolicyRepo()
}
