package components

import (
	"github.com/scipipe/scipipe"
	sp "github.com/scipipe/scipipe"
)

// NewIPSelectorSync returns a new IPSelectorSync component.  See the docs for
// IPSelectorSync for more information about how to configure and use it.
func NewIPSelectorSync(wf *sp.Workflow, name string, includeFunc func(ip *sp.FileIP) bool) *IPSelectorSync {
	p := &IPSelectorSync{
		BaseProcess: sp.NewBaseProcess(wf, name),
		includeFunc: includeFunc,
	}
	wf.AddProc(p)
	return p
}

// IPSelectorSync enables filtering IPs (FileIPs to be specific) by applying the
// supplied function includeFunc, which, if it returns true for an IP, will
// include it.
// The IPSelectorSync requires that the same number and names of ports are used and
// connected both for in-ports and out-ports. So, if you have an in-port
// 'data1', and 'data2', you need to create and connect also out-ports 'data1',
// and 'data2'.
// It will read all in-ports in a synchronous manner, and drop all IPs in the
// current iteration, if the condition in the includeFunc is not met.
type IPSelectorSync struct {
	sp.BaseProcess
	includeFunc func(*sp.FileIP) bool
}

// In returns an in-port if it exists, or creates it before, if it does not exist
func (p *IPSelectorSync) In(name string) *sp.InPort {
	if _, ok := p.InPorts()[name]; !ok {
		p.InitInPort(p, name)
	}
	return p.InPort(name)
}

// Out returns an out-port if it exists, or creates it before, if it does not exist
func (p *IPSelectorSync) Out(name string) *sp.OutPort {
	if _, ok := p.OutPorts()[name]; !ok {
		p.InitOutPort(p, name)
	}
	return p.OutPort(name)
}

// Run runs the component
func (p *IPSelectorSync) Run() {
	defer p.CloseAllOutPorts()

	for ips := range p.syncRead() {
		for _, ip := range ips {
			if !p.includeFunc(ip) {
				goto End
			}
		}
		for iname, ip := range ips {
			p.Out(iname).Send(ip) // Send on an out-port with the same name as the in-port
		}
	End:
		continue
	}
}

func (p *IPSelectorSync) syncRead() (ipSetChan chan map[string]*scipipe.FileIP) {
	ipSetChan = make(chan map[string]*scipipe.FileIP, 16)
	go func() {
		defer close(ipSetChan)
		for ips, ok := p.recvOneEach(); ok; ips, ok = p.recvOneEach() {
			ipSetChan <- ips
		}
	}()
	return ipSetChan
}

func (p *IPSelectorSync) recvOneEach() (ips map[string]*scipipe.FileIP, ok bool) {
	ips = make(map[string]*scipipe.FileIP)
	ok = true
	for inPortName, inPort := range p.InPorts() {
		ip, ok2 := <-inPort.Chan
		ips[inPortName] = ip
		ok = ok && ok2
	}
	return ips, ok
}
