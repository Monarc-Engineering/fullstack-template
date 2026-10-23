package handlers

import "github.com/gin-gonic/gin"

// EmailPreviewHandler serves the email template preview.
// Only registered in debug mode.
//
//	@Summary		Email template preview
//	@Description	Each email rendered with sample data, as Mailjet would get it. Local development only.
//	@Tags			observability
//	@Produce		html
//	@Param			email	query		string	false	"Sample to show"
//	@Param			view	query		string	false	"text for the plain-text part"
//	@Param			width	query		string	false	"phone for a 375px-wide frame"
//	@Success		200		{string}	string	"Preview page"
//	@Failure		404		{string}	string	"Unknown sample"
//	@Router			/dev/emails [get]
func (h *Handler) EmailPreviewHandler(_ *gin.Context) {}
