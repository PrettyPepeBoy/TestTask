package endpoint

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"testTask/internal/cast"
	"testTask/internal/parser"
)

var routingMap = map[string]route{
	"/status": {handler: func(ctx *fasthttp.RequestCtx, handler *HttpHandler) {
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBodyString("OK")
	}},

	"/api/v1/hab": {handler: func(ctx *fasthttp.RequestCtx, handler *HttpHandler) {
		method := cast.ByteArrayToSting(ctx.Method())
		if method == fasthttp.MethodDelete {
			handler.deleteHab(ctx)
		} else if method == fasthttp.MethodPut {
			handler.addHab(ctx)
		} else if method == fasthttp.MethodPost {
			handler.changeIntervalForHab(ctx)
		} else {
			ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
		}
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
}

func NewHttpHandler(parser *parser.Parser) *HttpHandler {
	return &HttpHandler{
		parser: parser,
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

func (h *HttpHandler) deleteHab(ctx *fasthttp.RequestCtx) {
	hab := cast.ByteArrayToSting(ctx.QueryArgs().Peek("hab"))
	err := h.parser.StopParsingHab(hab)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(err.Error())
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString(fmt.Sprintf("successfully stop parsing %s", hab))
}

func (h *HttpHandler) addHab(ctx *fasthttp.RequestCtx) {
	hab := cast.ByteArrayToSting(ctx.QueryArgs().Peek("hab"))
	err := h.parser.AddHabForParsing(hab)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(err.Error())
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString(fmt.Sprintf("successfully add for parsing %s", hab))
}

func (h *HttpHandler) changeIntervalForHab(ctx *fasthttp.RequestCtx) {
	hab := cast.ByteArrayToSting(ctx.QueryArgs().Peek("hab"))
	interval := cast.ByteArrayToSting(ctx.QueryArgs().Peek("duration"))
	err := h.parser.ChangeIntervalForHab(hab, interval)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBodyString(err.Error())
		return
	}

	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString(fmt.Sprintf("successfully change interval parsing for %s, to %s", hab, interval))
}
