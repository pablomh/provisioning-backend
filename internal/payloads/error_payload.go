package payloads

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/RHEnVision/provisioning-backend/internal/clients"
	httpClients "github.com/RHEnVision/provisioning-backend/internal/clients/http"
	"github.com/RHEnVision/provisioning-backend/internal/logging"
	"github.com/RHEnVision/provisioning-backend/internal/version"
	"github.com/go-chi/render"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ResponseError is used as a payload for all errors. Use NewResponseError function
// to create new type to set some fields correctly.
type ResponseError struct {
	// HTTP status code
	HTTPStatusCode int `json:"-" yaml:"-"`

	// user facing error message
	Message string `json:"msg,omitempty" yaml:"msg,omitempty"`

	// trace id from context (if provided)
	TraceId string `json:"trace_id,omitempty" yaml:"trace_id"`

	// edge id from context (if provided)
	EdgeId string `json:"edge_id,omitempty" yaml:"edge_id"`

	// full root cause
	Error string `json:"error" yaml:"error"`

	// build commit
	Version string `json:"version" yaml:"version"`

	// build time
	BuildTime string `json:"build_time" yaml:"build_time"`

	// environment (prod or stage or ephemeral)
	Environment string `json:"environment,omitempty" yaml:"environment"`
}

func (e *ResponseError) Render(_ http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

func NewResponseError(ctx context.Context, status int, userMsg string, err error) *ResponseError {
	var event *zerolog.Event
	var strError string

	if status < 500 {
		event = zerolog.Ctx(ctx).Warn().Stack()
	} else {
		event = zerolog.Ctx(ctx).Error().Stack()
	}
	if err != nil {
		event = event.Err(err)
		strError = err.Error()
	}
	if userMsg == "" {
		// take only part up to the first colon to avoid unique ids (UUIDs, database IDs etc)
		userMsg = strings.SplitN(err.Error(), ":", 2)[0]
	}
	event.Msg(userMsg)

	return &ResponseError{
		HTTPStatusCode: status,
		Message:        userMsg,
		TraceId:        logging.TraceId(ctx),
		Error:          strError,
		Version:        version.BuildCommit,
		BuildTime:      version.BuildTime,
	}
}

func NewInvalidRequestError(ctx context.Context, message string, err error) *ResponseError {
	message = fmt.Sprintf("Invalid request: %s", message)
	return NewResponseError(ctx, http.StatusBadRequest, message, err)
}

func NewWrongArchitectureUserError(ctx context.Context, err error) *ResponseError {
	return NewResponseError(ctx, http.StatusBadRequest, "Image and type architecture mismatch", err)
}

func NewMissingRequestParameterError(ctx context.Context, message string) *ResponseError {
	return NewResponseError(ctx, http.StatusBadRequest, message, nil)
}

func PubkeyDuplicateError(ctx context.Context, message string, err error) *ResponseError {
	return NewResponseError(ctx, http.StatusUnprocessableEntity, message, err)
}

type userPayload struct {
	code    int
	message string
}

var errStatus = map[error]*userPayload{
	// generic errors
	clients.HttpClientErr:     {500, "unknown backend client error"},
	clients.BadRequestErr:     {400, "bad request; returned from a backend service"},
	clients.NotFoundErr:       {404, "not found; returned from a backend service"},
	clients.UnauthorizedErr:   {401, "unauthorized; returned from a backend service"},
	clients.ForbiddenErr:      {403, "forbidden; returned from a backend service"},
	clients.Non2xxResponseErr: {500, "unsuccessful response;returned from a backend service"},

	// image builder specific errors
	httpClients.CloneNotFoundErr:        {404, "image builder could not find compose clone"},
	httpClients.ComposeNotFoundErr:      {404, "image builder could not find compose"},
	httpClients.ImageStatusErr:          {400, "image builder compose not successfully built"},
	httpClients.UnknownImageTypeErr:     {400, "wrong type of image builder compose"},
	httpClients.UploadStatusErr:         {400, "wrong compose status of image builder compose"},
	httpClients.ImageRequestNotFoundErr: {404, "image builder compose request not found"},

	// sources specific errors
	clients.UnknownAuthenticationTypeErr: {500, "unknown authentication type"},
	clients.UnknownProviderErr:           {500, "unknown provider type"},
	clients.MissingProvisioningSources:   {500, "backend service missing provisioning source"},
	httpClients.NotEvenErr:               {500, "client arguments error"},
}

func findUserPayload(err error) *userPayload {
	if err == nil {
		return nil
	}

	if result, ok := errStatus[err]; ok {
		return result
	}

	return findUserPayload(errors.Unwrap(err))
}

func NewClientError(ctx context.Context, err error) *ResponseError {
	if payload := findUserPayload(err); payload != nil {
		logger := log.Ctx(ctx).Warn()
		if payload.code >= 500 {
			logger = log.Ctx(ctx).Error()
		}
		logger.Msgf("Client error: %s", err)
		return NewResponseError(ctx, payload.code, payload.message, err)
	}
	log.Ctx(ctx).Error().Msgf("Unknown client error: %s", err)
	return NewResponseError(ctx, 500, "backend client error", err)
}

func NewNotFoundError(ctx context.Context, message string, err error) *ResponseError {
	message = fmt.Sprintf("Not found: %s", message)
	return NewResponseError(ctx, http.StatusNotFound, message, err)
}

func NewEnqueueTaskError(ctx context.Context, message string, err error) *ResponseError {
	message = fmt.Sprintf("Task enqueue error: %s", message)
	return NewResponseError(ctx, http.StatusInternalServerError, message, err)
}

func NewDAOError(ctx context.Context, message string, err error) *ResponseError {
	message = fmt.Sprintf("DAO error: %s", message)
	return NewResponseError(ctx, http.StatusInternalServerError, message, err)
}

func NewRenderError(ctx context.Context, message string, err error) *ResponseError {
	message = fmt.Sprintf("Rendering error: %s", message)
	return NewResponseError(ctx, http.StatusInternalServerError, message, err)
}

func NewURLParsingError(ctx context.Context, message string, err error) *ResponseError {
	message = fmt.Sprintf("URL parsing error: %s", message)
	return NewResponseError(ctx, http.StatusBadRequest, message, err)
}

func NewStatusError(ctx context.Context, message string, err error) *ResponseError {
	message = fmt.Sprintf("Status error: %s", message)
	return NewResponseError(ctx, http.StatusInternalServerError, message, err)
}

func NewAWSError(ctx context.Context, message string, err error) *ResponseError {
	message = fmt.Sprintf("AWS API error: %s", message)
	return NewResponseError(ctx, http.StatusInternalServerError, message, err)
}

func NewAzureError(ctx context.Context, message string, err error) *ResponseError {
	message = fmt.Sprintf("Azure API error: %s", message)
	return NewResponseError(ctx, http.StatusInternalServerError, message, err)
}

func NewGCPError(ctx context.Context, message string, err error) *ResponseError {
	message = fmt.Sprintf("Google API error: %s", message)
	return NewResponseError(ctx, http.StatusInternalServerError, message, err)
}
