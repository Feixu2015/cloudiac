package apps

import (
	"cloudiac/portal/consts/e"
	"cloudiac/portal/libs/ctx"
	"cloudiac/portal/models"
	"cloudiac/portal/models/forms"
	"cloudiac/portal/services"
	"cloudiac/utils"
	"cloudiac/utils/mail"
	"fmt"
	"net/http"
)

var (
	emailSubjectResetPassword = "重置密码"
	emailBodyResetPassword    = "尊敬的 {{.Name}}：\n\n您的密码已经被重置，这是您的新密码：\n\n密码：\t{{.InitPass}}\n\n请使用新密码登陆系统。\n\n为了保障您的安全，请立即登陆您的账号并修改密码。"
)

func SearchToken(c *ctx.ServiceCtx, form *forms.SearchTokenForm) (interface{}, e.Error) {
	//todo 鉴权
	query := services.QueryToken(c.DB())
	query = query.Where("org_id = ?", c.OrgId)
	if form.Status != "" {
		query = query.Where("status = ?", form.Status)
	}
	if form.Q != "" {
		qs := "%" + form.Q + "%"
		query = query.Where("description LIKE ?", qs)
	}

	query = query.Order("created_at DESC")
	rs, err := getPage(query, form, models.Token{})
	if err != nil {
		c.Logger().Errorf("error get page, err %s", err)
		return nil, err
	}
	return rs, nil
}

func CreateToken(c *ctx.ServiceCtx, form *forms.CreateTokenForm) (interface{}, e.Error) {
	c.AddLogField("action", fmt.Sprintf("create token for user %s", c.UserId))

	tokenStr := utils.GenGuid("")
	token, err := services.CreateToken(c.DB(), models.Token{
		Key:         tokenStr,
		Type:        form.Type,
		OrgId:       c.OrgId,
		Role:        form.Role,
		ExpiredAt:   form.ExpiredAt,
		Description: form.Description,
	})
	if err != nil && err.Code() == e.TokenAlreadyExists {
		return nil, e.New(err.Code(), err, http.StatusBadRequest)
	} else if err != nil {
		c.Logger().Errorf("error creating token, err %s", err)
		return nil, e.AutoNew(err, e.DBError)
	}

	return token, nil
}

func UpdateToken(c *ctx.ServiceCtx, form *forms.UpdateTokenForm) (token *models.Token, err e.Error) {
	c.AddLogField("action", fmt.Sprintf("update token %s", form.Id))
	if form.Id == "" {
		return nil, e.New(e.BadRequest, fmt.Errorf("missing 'id'"))
	}

	attrs := models.Attrs{}
	if form.HasKey("status") {
		attrs["status"] = form.Status
	}

	if form.HasKey("description") {
		attrs["description"] = form.Description
	}

	token, err = services.UpdateToken(c.DB(), form.Id, attrs)
	if err != nil && err.Code() == e.TokenAliasDuplicate {
		return nil, e.New(err.Code(), err, http.StatusBadRequest)
	} else if err != nil {
		c.Logger().Errorf("error update org, err %s", err)
		return nil, err
	}
	return
}

func DeleteToken(c *ctx.ServiceCtx, form *forms.DeleteTokenForm) (result interface{}, re e.Error) {
	c.AddLogField("action", fmt.Sprintf("delete token %s", form.Id))
	if err := services.DeleteToken(c.DB(), form.Id); err != nil {
		return nil, err
	}

	return
}

// UserPassReset 用户重置密码
func UserPassReset(c *ctx.ServiceCtx, form *forms.DetailUserForm) (*models.User, e.Error) {
	initPass := utils.GenPasswd(6, "mix")
	hashedPassword, err := services.HashPassword(initPass)
	if err != nil {
		c.Logger().Errorf("error hash password %s", err)
		return nil, err
	}

	attrs := models.Attrs{}
	attrs["init_pass"] = initPass
	attrs["password"] = hashedPassword

	user, err := services.UpdateUser(c.DB(), form.Id, attrs)

	resp := struct {
		*models.User
		InitPass string
	}{
		User:     user,
		InitPass: initPass,
	}

	// TODO: 需确定邮件内容
	go func() {
		err := mail.SendMail([]string{user.Email}, emailSubjectResetPassword, utils.SprintTemplate(emailBodyResetPassword, resp))
		if err != nil {
			c.Logger().Errorf("error send mail to %s, err %s", user.Email, err)
		}
	}()

	return user, err
}
