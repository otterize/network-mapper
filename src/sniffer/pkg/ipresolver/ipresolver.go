package ipresolver

type IPResolver interface {
	Refresh() error
	ResolveIP(ipaddr string) (hostname string, err error)
}

type MockIPResolver struct{}

func (r *MockIPResolver) Refresh() error { return nil }
func (r *MockIPResolver) ResolveIP(_ string) (hostname string, err error) {
	return "", nil
}
