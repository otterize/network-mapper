package ipresolver

type IPResolver interface {
	WaitForNextRefresh()
	ResolveIP(ipaddr string) (hostname string, err error)
}
