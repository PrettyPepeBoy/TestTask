package endpoint

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"testTask/internal/cast"
	"testTask/internal/parser"
	"testTask/internal/user"
)

var ErrNoTokenProvided = errors.New("no token provided")

var routingMap = map[string]route{
	"/status": {handler: func(ctx *fasthttp.RequestCtx, handler *HttpHandler) {
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBodyString("OK")
	}},

	"/api/v1/parse": {handler: func(ctx *fasthttp.RequestCtx, handler *HttpHandler) {
		method := cast.ByteArrayToSting(ctx.Method())
		if method == fasthttp.MethodDelete {
			handler.stopParseHab(ctx)
		} else if method == fasthttp.MethodPut {
			handler.addHab(ctx)
		} else if method == fasthttp.MethodPost {
			handler.changeIntervalForHab(ctx)
		} else {
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		}
	}},

	"/api/v1/hab": {handler: func(ctx *fasthttp.RequestCtx, handler *HttpHandler) {

	}},
}

func init() {
	for path, r := range routingMap {
		r.path = path
		routingMap[path] = r
	}
}

type route struct {
	path    string
	handler func(ctx *fasthttp.RequestCtx, handler *HttpHandler)
}

type HttpHandler struct {
	parser *parser.Parser
	auth   *user.Authorizer
}

func NewHttpHandler(parser *parser.Parser, auth *user.Authorizer) *HttpHandler {
	return &HttpHandler{
		parser: parser,
		auth:   auth,
	}
}

func (h *HttpHandler) Handle(ctx *fasthttp.RequestCtx) {
	defer func() {
		err := recover()
		if err != nil {
			logrus.Error("Critical error during handling: ", err)
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		}
	}()

	if r, ok := routingMap[cast.ByteArrayToSting(ctx.Path())]; ok {
		r.handler(ctx, h)
	} else {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
	}
}

func (h *HttpHandler) stopParseHab(ctx *fasthttp.RequestCtx) {
	_, err := h.authorizeModification(ctx)
	if err != nil {
		writeError(ctx, err.Error(), fasthttp.StatusForbidden)
		return
	}

	hab := cast.ByteArrayToSting(ctx.QueryArgs().Peek("hab"))

	err = h.parser.StopParsingHab(hab)
	if err != nil {
		writeError(ctx, err.Error(), fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString(fmt.Sprintf("successfully stop parsing %s", hab))
}

func (h *HttpHandler) addHab(ctx *fasthttp.RequestCtx) {
	_, err := h.authorizeModification(ctx)
	if err != nil {
		writeError(ctx, err.Error(), fasthttp.StatusForbidden)
		return
	}

	hab := cast.ByteArrayToSting(ctx.QueryArgs().Peek("hab"))

	err = h.parser.AddHabForParsing(hab)
	if err != nil {
		writeError(ctx, err.Error(), fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString(fmt.Sprintf("successfully add for parsing %s", hab))
}

func (h *HttpHandler) changeIntervalForHab(ctx *fasthttp.RequestCtx) {
	_, err := h.authorizeModification(ctx)
	if err != nil {
		writeError(ctx, err.Error(), fasthttp.StatusForbidden)
		return
	}

	hab := cast.ByteArrayToSting(ctx.QueryArgs().Peek("hab"))
	interval := cast.ByteArrayToSting(ctx.QueryArgs().Peek("duration"))

	err = h.parser.ChangeIntervalForHab(hab, interval)
	if err != nil {
		writeError(ctx, err.Error(), fasthttp.StatusBadRequest)
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString(fmt.Sprintf("successfully change interval parsing for %s, to %s", hab, interval))
}

func (h *HttpHandler) authorizeModification(ctx *fasthttp.RequestCtx) (string, error) {
	token := ctx.Request.Header.Peek("Private-Token")
	if len(token) == 0 {
		return "", ErrNoTokenProvided
	}

	return h.auth.Verify(cast.ByteArrayToSting(token))
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(ctx *fasthttp.RequestCtx, message string, status int) {
	response := errorResponse{Error: message}
	row, err := json.Marshal(&response)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	ctx.SetStatusCode(status)
	ctx.Response.Header.Set(fasthttp.HeaderContentType, "application/json")
	_, _ = ctx.Write(row)
}
