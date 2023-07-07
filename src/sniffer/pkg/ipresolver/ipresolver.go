package ipresolver

type IPResolver interface {
	WaitForNextRefresh()
	ResolveIP(ipaddr string) (hostname string, err error)
}

type MockIPResolver struct{}

func (r *MockIPResolver) WaitForNextRefresh() {}
func (r *MockIPResolver) ResolveIP(_ string) (hostname string, err error) {
	return "", nil
}
