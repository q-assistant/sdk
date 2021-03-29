package event

type Handler struct {
	On      string
	Handler HandlerFunc
}

type HandlerFunc func()
