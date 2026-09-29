package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"justixauto/internal/modules/identity/service"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/httpx"
)

// CompanyUserHandler lets a company admin manage the company's employees
// with roles prepared in Admin (user decisions 2026-09-26).
type CompanyUserHandler struct {
	users *service.CompanyUser
}

// NewCompanyUserHandler builds the company employee handler.
func NewCompanyUserHandler(users *service.CompanyUser) *CompanyUserHandler {
	return &CompanyUserHandler{users: users}
}

type passwordBody struct {
	Password string `json:"password"`
}

// Routes: the service checks membership and company.users.manage.
func (h *CompanyUserHandler) Routes(g *echo.Group) {
	g.GET("/companies/:id/roles", h.roles, auth.Require())
	g.POST("/companies/:id/roles", h.createRole, auth.Require())
	g.PATCH("/companies/:id/roles/:roleId", h.updateRole, auth.Require())
	g.POST("/companies/:id/roles/:roleId/delete", h.deleteRole, auth.Require())
	g.GET("/companies/:id/permissions", h.permissions, auth.Require())
	g.GET("/companies/:id/users", h.list, auth.Require())
	g.POST("/companies/:id/users", h.create, auth.Require())
	g.PATCH("/companies/:id/users/:userId", h.update, auth.Require())
	g.POST("/companies/:id/users/:userId/password", h.setPassword, auth.Require())
	g.POST("/companies/:id/users/:userId/suspend", h.suspend, auth.Require())
	g.POST("/companies/:id/users/:userId/restore", h.restore, auth.Require())
}

// roles lists the roles the company may assign: the built-in company
// administrator and the company's own roles.
//
//	@Summary	Company roles
//	@Tags		identity/company-users
//	@Param		id			path		string	true	"company ID"
//	@Success	200			{object}	httpx.ListEnvelope[handler.roleDTO]
//	@Failure	401,403,404	{object}	httpx.ErrorBody
//	@Router		/identity/companies/{id}/roles [get]
func (h *CompanyUserHandler) roles(c *echo.Context) error {
	roles, err := h.users.Roles(c.Request().Context(), auth.Get(c), c.Param("id"))
	if err != nil {
		return err
	}
	return httpx.List(c, mapSlice(roles, toRole), nil)
}

// permissions lists the company permissions the company's roles may hold.
//
//	@Summary	Company permissions
//	@Tags		identity/company-users
//	@Param		id			path		string	true	"company ID"
//	@Success	200			{object}	httpx.ListEnvelope[handler.permissionDTO]
//	@Failure	401,403,404	{object}	httpx.ErrorBody
//	@Router		/identity/companies/{id}/permissions [get]
func (h *CompanyUserHandler) permissions(c *echo.Context) error {
	perms, err := h.users.Permissions(c.Request().Context(), auth.Get(c), c.Param("id"))
	if err != nil {
		return err
	}
	return httpx.List(c, mapSlice(perms, toPermission), nil)
}

// createRole adds a private role of the company.
//
//	@Summary	Create company role
//	@Tags		identity/company-users
//	@Security	CSRF
//	@Param		id					path		string				true	"company ID"
//	@Param		body				body		service.RoleInput	true	"role"
//	@Success	201					{object}	httpx.DataEnvelope[handler.roleDTO]
//	@Failure	401,403,404,409,422	{object}	httpx.ErrorBody
//	@Router		/identity/companies/{id}/roles [post]
func (h *CompanyUserHandler) createRole(c *echo.Context) error {
	var in service.RoleInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	r, err := h.users.CreateRole(c.Request().Context(), auth.Get(c), c.Param("id"), in)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusCreated, toRole(r), r.Version)
}

// updateRole edits one of the company's own roles.
//
//	@Summary	Update company role
//	@Tags		identity/company-users
//	@Security	CSRF
//	@Param		id							path		string				true	"company ID"
//	@Param		roleId						path		string				true	"role ID"
//	@Param		If-Match					header		string				true	"revision"
//	@Param		body						body		service.RoleInput	true	"role"
//	@Success	200							{object}	httpx.DataEnvelope[handler.roleDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/identity/companies/{id}/roles/{roleId} [patch]
func (h *CompanyUserHandler) updateRole(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in service.RoleInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	r, err := h.users.UpdateRole(c.Request().Context(), auth.Get(c), c.Param("id"), c.Param("roleId"), expected, in)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toRole(r), r.Version)
}

// deleteRole soft-deletes one of the company's own roles.
//
//	@Summary	Delete company role
//	@Tags		identity/company-users
//	@Security	CSRF
//	@Param		id			path	string	true	"company ID"
//	@Param		roleId		path	string	true	"role ID"
//	@Param		If-Match	header	string	true	"revision"
//	@Success	204
//	@Failure	401,403,404,409,412,428	{object}	httpx.ErrorBody
//	@Router		/identity/companies/{id}/roles/{roleId}/delete [post]
func (h *CompanyUserHandler) deleteRole(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	if err := h.users.DeleteRole(c.Request().Context(), auth.Get(c), c.Param("id"), c.Param("roleId"), expected); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// list lists the company's employees.
//
//	@Summary	List company employees
//	@Tags		identity/company-users
//	@Param		id			path		string	true	"company ID"
//	@Success	200			{object}	httpx.ListEnvelope[handler.userDTO]
//	@Failure	401,403,404	{object}	httpx.ErrorBody
//	@Router		/identity/companies/{id}/users [get]
func (h *CompanyUserHandler) list(c *echo.Context) error {
	users, err := h.users.List(c.Request().Context(), auth.Get(c), c.Param("id"))
	if err != nil {
		return err
	}
	return httpx.List(c, mapSlice(users, toUser), nil)
}

// create adds an employee with login, temporary password and prepared roles.
//
//	@Summary	Add company employee
//	@Tags		identity/company-users
//	@Security	CSRF
//	@Param		id					path		string						true	"company ID"
//	@Param		body				body		service.CompanyUserInput	true	"employee"
//	@Success	201					{object}	httpx.DataEnvelope[handler.userDTO]
//	@Failure	401,403,404,409,422	{object}	httpx.ErrorBody
//	@Router		/identity/companies/{id}/users [post]
func (h *CompanyUserHandler) create(c *echo.Context) error {
	var in service.CompanyUserInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	u, err := h.users.Create(c.Request().Context(), auth.Get(c), c.Param("id"), in)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusCreated, toUser(u), u.User.Version)
}

// update changes an employee's name, email and roles.
//
//	@Summary	Update company employee
//	@Tags		identity/company-users
//	@Security	CSRF
//	@Param		id							path		string							true	"company ID"
//	@Param		userId						path		string							true	"user ID"
//	@Param		If-Match					header		string							true	"revision"
//	@Param		body						body		service.UpdateCompanyUserInput	true	"employee"
//	@Success	200							{object}	httpx.DataEnvelope[handler.userDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/identity/companies/{id}/users/{userId} [patch]
func (h *CompanyUserHandler) update(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in service.UpdateCompanyUserInput
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	u, err := h.users.Update(c.Request().Context(), auth.Get(c), c.Param("id"), c.Param("userId"), expected, in)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toUser(u), u.User.Version)
}

// setPassword gives an employee a new temporary password.
//
//	@Summary	Reset company employee password
//	@Tags		identity/company-users
//	@Security	CSRF
//	@Param		id							path		string			true	"company ID"
//	@Param		userId						path		string			true	"user ID"
//	@Param		If-Match					header		string			true	"revision"
//	@Param		body						body		passwordBody	true	"temporary password"
//	@Success	200							{object}	httpx.DataEnvelope[handler.userDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/identity/companies/{id}/users/{userId}/password [post]
func (h *CompanyUserHandler) setPassword(c *echo.Context) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in passwordBody
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	u, err := h.users.SetPassword(c.Request().Context(), auth.Get(c), c.Param("id"), c.Param("userId"), expected, in.Password)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toUser(u), u.User.Version)
}

// suspend blocks an employee's sign-in; history is kept.
//
//	@Summary	Suspend company employee
//	@Tags		identity/company-users
//	@Security	CSRF
//	@Param		id							path		string		true	"company ID"
//	@Param		userId						path		string		true	"user ID"
//	@Param		If-Match					header		string		true	"revision"
//	@Param		body						body		reasonBody	true	"reason"
//	@Success	200							{object}	httpx.DataEnvelope[handler.userDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/identity/companies/{id}/users/{userId}/suspend [post]
func (h *CompanyUserHandler) suspend(c *echo.Context) error { return h.setStatus(c, true) }

// restore re-enables a suspended employee.
//
//	@Summary	Restore company employee
//	@Tags		identity/company-users
//	@Security	CSRF
//	@Param		id							path		string		true	"company ID"
//	@Param		userId						path		string		true	"user ID"
//	@Param		If-Match					header		string		true	"revision"
//	@Param		body						body		reasonBody	true	"reason"
//	@Success	200							{object}	httpx.DataEnvelope[handler.userDTO]
//	@Failure	401,403,404,409,412,422,428	{object}	httpx.ErrorBody
//	@Router		/identity/companies/{id}/users/{userId}/restore [post]
func (h *CompanyUserHandler) restore(c *echo.Context) error { return h.setStatus(c, false) }

func (h *CompanyUserHandler) setStatus(c *echo.Context, suspend bool) error {
	expected, err := httpx.IfMatch(c)
	if err != nil {
		return err
	}
	var in reasonBody
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	u, err := h.users.SetStatus(c.Request().Context(), auth.Get(c), c.Param("id"), c.Param("userId"), expected, in.Reason, suspend)
	if err != nil {
		return err
	}
	return httpx.Data(c, http.StatusOK, toUser(u), u.User.Version)
}
