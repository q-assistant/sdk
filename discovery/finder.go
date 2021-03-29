package discovery

type Finder struct {
	client Discovery
}

func NewFinder(client Discovery) *Finder {
	return &Finder{client: client}
}

func (f *Finder) Find(service string) ([]*Service, error) {
	return f.client.Find(service)
}
